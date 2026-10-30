#!/usr/bin/env python3
"""复制现有 GitHub Release 到 GitLab；只依赖 Python 标准库和 Git。

用法：GITLAB_TOKEN=<项目令牌> GH_TOKEN=<GitHub令牌，可选> \
    python3 deploy/mirror-gitlab-release.py vX.Y.Z
发行包必须来自 GitHub，不重新构建；所有 Git 操作均在临时 bare 仓库执行。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
from urllib.error import HTTPError
from urllib.parse import quote, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener


GITHUB_API = "https://api.github.com/repos/mcyudream/YPanel"
GITLAB_URL = "https://gitlab.yudream.online"
PROJECT_URL = f"{GITLAB_URL}/yudream/ypanel"
GITLAB_API = f"{GITLAB_URL}/api/v4/projects/yudream%2Fypanel"
PACKAGES = ("ypanel-linux-amd64.tar.gz", "ypanel-linux-arm64.tar.gz")
FILES = (*PACKAGES, "sha256sums.txt")


class SafeRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        redirected = super().redirect_request(req, fp, code, msg, headers, newurl)
        if redirected and urlsplit(req.full_url).netloc != urlsplit(newurl).netloc:
            # GitHub 资产会跳转到对象存储，不向下载主机转发 API 令牌。
            redirected.remove_header("Authorization")
            redirected.remove_header("Private-token")
        return redirected


HTTP = build_opener(SafeRedirect())


def api(url, headers, method="GET", payload=None, allow_missing=False):
    body = None if payload is None else json.dumps(payload).encode()
    request = Request(
        url,
        data=body,
        headers={**headers, "Content-Type": "application/json"},
        method=method,
    )
    try:
        with HTTP.open(request, timeout=60) as response:
            return json.load(response)
    except HTTPError as error:
        if allow_missing and error.code == 404:
            return None
        # 不打印可能包含密钥或用户数据的响应正文。
        raise RuntimeError(f"{method} {urlsplit(url).path}: HTTP {error.code}") from None


def download(url, path, headers=None):
    request = Request(url, headers=headers or {}, method="GET")
    try:
        with HTTP.open(request, timeout=120) as response, path.open("wb") as output:
            while block := response.read(1024 * 1024):
                output.write(block)
    except HTTPError as error:
        raise RuntimeError(f"下载 {path.name}: HTTP {error.code}") from None


def file_digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def verify_packages(directory):
    sums = {}
    for line in (directory / "sha256sums.txt").read_text().splitlines():
        match = re.fullmatch(r"([0-9a-fA-F]{64}) [ *](\S+)", line)
        if not match or match[2] not in PACKAGES or match[2] in sums:
            raise RuntimeError("sha256sums.txt 包含无效、重复或非预期的文件名")
        sums[match[2]] = match[1].lower()
    if set(sums) != set(PACKAGES):
        raise RuntimeError("sha256sums.txt 缺少双架构发行包的摘要")
    for name, digest in sums.items():
        if file_digest(directory / name) != digest:
            raise RuntimeError(f"GitHub 发行包 SHA-256 不匹配：{name}")
    return {name: file_digest(directory / name) for name in FILES}


def sync_refs(directory, tag, expected_commit):
    """临时仓库构造资产树，普通推送保留现有引用，不 checkout 源码工作区。"""
    repo = directory / "mirror.git"
    subprocess.run(["git", "init", "--bare", "--quiet", str(repo)], check=True)
    helper = directory / "gitlab-credential-helper"
    helper.write_text(
        '#!/bin/sh\nif [ "$1" = get ]; then\n'
        '  printf "username=oauth2\\npassword=%s\\n" "$GITLAB_TOKEN"\nfi\n'
    )
    helper.chmod(0o700)
    env = {
        **os.environ,
        "GIT_TERMINAL_PROMPT": "0",
        "GIT_AUTHOR_NAME": "YuDream Release",
        "GIT_AUTHOR_EMAIL": "release@users.noreply.github.com",
        "GIT_COMMITTER_NAME": "YuDream Release",
        "GIT_COMMITTER_EMAIL": "release@users.noreply.github.com",
    }

    def git(*args, stdin=None):
        command = [
            "git", "-c", "credential.helper=",
            "-c", f"credential.{GITLAB_URL}.helper={helper}",
            # GitLab 入口的大包 HTTP/2 上传实测触发 curl 92 PROTOCOL_ERROR。
            "-c", f"http.{GITLAB_URL}/.version=HTTP/1.1",
            # 两个发行包合并为单次 Git 推送；固定 Content-Length 避免代理卡住 chunked 上传。
            "-c", f"http.{GITLAB_URL}/.postBuffer=268435456",
            "-c", f"http.{GITLAB_URL}/.lowSpeedLimit=1024",
            "-c", f"http.{GITLAB_URL}/.lowSpeedTime=90",
            # 单次别名绕开开发机 github.com 的全局下载代理重写。
            "-c", "url.https://github.com/.insteadOf=ypanel-github:",
            *args,
        ]
        result = subprocess.run(
            command, cwd=repo, env=env, input=stdin,
            capture_output=True, text=True, timeout=600,
        )
        if result.returncode:
            message = result.stderr.strip()
            for name in ("GITLAB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"):
                if value := os.environ.get(name):
                    message = message.replace(value, "[REDACTED]")
            raise RuntimeError(f"git {args[0]} 失败：{message}")
        return result.stdout.strip()

    git(
        "fetch", "--quiet", "--no-tags", "ypanel-github:mcyudream/YPanel.git",
        "refs/heads/main:refs/heads/main", f"refs/tags/{tag}:refs/tags/{tag}",
    )
    if git("rev-parse", f"refs/tags/{tag}^{{commit}}") != expected_commit:
        raise RuntimeError("GitHub 标签的实际提交与 API 返回不一致，停止同步")

    entries = []
    for name in sorted(FILES):
        blob = git("hash-object", "-w", "--no-filters", str(directory / name))
        entries.append(f"100644 blob {blob}\t{name}\n")
    tree = git("mktree", stdin="".join(entries))
    branch = f"refs/heads/assets/{tag}"
    remote = f"{PROJECT_URL}.git"
    if git("ls-remote", "--heads", remote, branch):
        git("fetch", "--quiet", "--no-tags", remote, f"{branch}:{branch}")
        if git("rev-parse", f"{branch}^{{tree}}") != tree:
            raise RuntimeError(f"GitLab assets/{tag} 已存在不同资产，停止覆盖")
        asset_commit = git("rev-parse", branch)
    else:
        asset_commit = git("commit-tree", tree, "-m", f"YPanel {tag} release assets")
    # 原子、非强制推送；目标有独立改动时失败，不能覆盖源码或已有版本。
    git(
        "push", "--atomic", remote, "refs/heads/main:refs/heads/main",
        f"refs/tags/{tag}:refs/tags/{tag}", f"{asset_commit}:{branch}",
    )
    print(f"GitLab 源码、标签与 assets/{tag} 已同步", flush=True)


def sync_release(tag, release, commit, headers):
    release_url = f"{GITLAB_API}/releases/{quote(tag, safe='')}"
    current = api(release_url, headers, allow_missing=True)
    links = [
        {
            "name": name,
            "url": f"{PROJECT_URL}/-/raw/assets/{tag}/{name}",
            "direct_asset_path": f"/{name}",
            "link_type": "package",
        }
        for name in FILES
    ]
    metadata = {
        "name": release.get("name") or f"YPanel {tag}",
        "description": release.get("body") or f"YPanel {tag} — synced from GitHub",
        "released_at": release["published_at"],
    }
    if current is None:
        api(f"{GITLAB_API}/releases", headers, "POST", {
            **metadata, "tag_name": tag, "ref": commit, "assets": {"links": links},
        })
    else:
        api(release_url, headers, "PUT", metadata)
        existing = {link["name"]: link for link in current["assets"]["links"]}
        for link in links:
            previous = existing.get(link["name"])
            endpoint = f"{release_url}/assets/links"
            direct_url = f"{PROJECT_URL}/-/releases/{tag}/downloads/{link['name']}"
            if previous is None:
                api(endpoint, headers, "POST", link)
            elif previous["url"] != link["url"] or previous.get("direct_asset_url") != direct_url:
                api(f"{endpoint}/{previous['id']}", headers, "PUT", link)


def verify_mirror(directory, tag, commit, digests, is_latest):
    # 验收必须匿名进行：API 可写并不代表用户能下载。
    remote_tag = api(f"{GITLAB_API}/repository/tags/{quote(tag, safe='')}", {})
    if remote_tag["commit"]["id"] != commit:
        raise RuntimeError("GitLab 标签提交与 GitHub 不一致")
    release = api(f"{GITLAB_API}/releases/{quote(tag, safe='')}", {})
    links = {link["name"]: link for link in release["assets"]["links"]}
    for name in FILES:
        expected_url = f"{PROJECT_URL}/-/releases/{tag}/downloads/{name}"
        if name not in links or links[name].get("direct_asset_url") != expected_url:
            raise RuntimeError(f"GitLab Release 缺少正确的下载直链：{name}")
        target = directory / f"verify-{name}"
        download(expected_url, target)
        if file_digest(target) != digests[name]:
            raise RuntimeError(f"GitLab 匿名下载 SHA-256 不匹配：{name}")
        print(f"匿名下载校验通过：{name} ({target.stat().st_size} bytes)", flush=True)
    if is_latest:
        latest = api(f"{GITLAB_API}/releases/permalink/latest", {})
        if latest["tag_name"] != tag:
            raise RuntimeError(f"GitLab 最新发行版仍是 {latest['tag_name']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="现有 GitHub Release 标签，例如 v0.9.43")
    args = parser.parse_args()
    if not re.fullmatch(r"v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?", args.tag):
        parser.error("版本标签必须为 vX.Y.Z（可含预发布后缀）")
    token = os.environ.get("GITLAB_TOKEN", "")
    if not token:
        raise RuntimeError("缺少 GITLAB_TOKEN，请配置本项目的 api + write_repository 发布令牌")
    github_headers = {"Accept": "application/vnd.github+json"}
    if github_token := os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN"):
        github_headers["Authorization"] = f"Bearer {github_token}"
    gitlab_headers = {"PRIVATE-TOKEN": token}
    # 在下载大文件或推送 Git 引用之前验证 API 权限，账号密码不能代替 API 令牌。
    api(f"{GITLAB_URL}/api/v4/user", gitlab_headers)
    release = api(f"{GITHUB_API}/releases/tags/{quote(args.tag, safe='')}", github_headers)
    if release.get("draft") or not release.get("published_at"):
        raise RuntimeError("只能镜像已发布的 GitHub Release")
    assets = {asset["name"]: asset for asset in release["assets"]}
    if not set(FILES).issubset(assets):
        raise RuntimeError("GitHub Release 缺少双架构发行包或 sha256sums.txt")
    commit = api(f"{GITHUB_API}/commits/{quote(args.tag, safe='')}", github_headers)["sha"]
    latest = api(f"{GITHUB_API}/releases/latest", github_headers)
    with tempfile.TemporaryDirectory(prefix="ypanel-gitlab-mirror-") as tmp:
        directory = Path(tmp)
        for name in FILES:
            asset = assets[name]
            print(f"下载 GitHub 已发布资产：{name}", flush=True)
            download(
                asset["url"], directory / name,
                {**github_headers, "Accept": "application/octet-stream"},
            )
            if (directory / name).stat().st_size != asset["size"]:
                raise RuntimeError(f"GitHub 资产下载大小不匹配：{name}")
            if digest := asset.get("digest"):
                if digest != f"sha256:{file_digest(directory / name)}":
                    raise RuntimeError(f"GitHub 资产 API 摘要不匹配：{name}")
        digests = verify_packages(directory)
        sync_refs(directory, args.tag, commit)
        sync_release(args.tag, release, commit, gitlab_headers)
        verify_mirror(directory, args.tag, commit, digests, latest["tag_name"] == args.tag)
    print(f"GitLab 镜像完成：{PROJECT_URL}/-/releases/{args.tag}")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f"GitLab 镜像失败：{error}", file=sys.stderr)
        sys.exit(1)
