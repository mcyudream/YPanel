package service

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name         string
		pk           DockerInstallPrecheck
		wantSupport  bool
		wantAction   string
		wantContains string
	}{
		{"ubuntu 全新完整安装", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "ubuntu", Version: "22.04", Codename: "jammy"}, true, "full", ""},
		{"debian 全新完整安装", DockerInstallPrecheck{Systemd: true, Arch: "aarch64", Distro: "debian", Version: "12", Codename: "bookworm"}, true, "full", ""},
		{"已装全绪无需安装", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "ubuntu", Codename: "jammy", DockerInstalled: true, ComposeInstalled: true}, true, "none", ""},
		{"仅缺 compose 补装", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "ubuntu", Codename: "jammy", DockerInstalled: true}, true, "compose-only", ""},
		{"缺 codename 不支持", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "debian", Version: "12"}, false, "", "发行版代号"},
		{"centos8 不支持", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "centos", Version: "8.5"}, false, "", "CentOS 8"},
		{"centos7 支持但标注 EOL", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "centos", Version: "7.9"}, true, "full", "EOL"},
		{"rocky9 支持", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "rocky", Version: "9.4"}, true, "full", ""},
		{"alinux3 映射 rhel8", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "alinux", Version: "3"}, true, "full", ""},
		{"openEuler 不支持", DockerInstallPrecheck{Systemd: true, Arch: "x86_64", Distro: "openEuler", Version: "22.03"}, false, "", "不支持的发行版"},
		{"无 systemd 不支持", DockerInstallPrecheck{Systemd: false, Arch: "x86_64", Distro: "ubuntu", Codename: "jammy"}, false, "", "systemd"},
		{"arm64 支持", DockerInstallPrecheck{Systemd: true, Arch: "aarch64", Distro: "rocky", Version: "9"}, true, "full", ""},
		{"riscv 不支持", DockerInstallPrecheck{Systemd: true, Arch: "riscv64", Distro: "ubuntu", Codename: "jammy"}, false, "", "架构"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pk := tc.pk
			classify(&pk)
			if pk.Supported != tc.wantSupport {
				t.Fatalf("supported = %v (reason=%s)", pk.Supported, pk.Reason)
			}
			if tc.wantAction != "" && pk.Action != tc.wantAction {
				t.Fatalf("action = %q, want %q", pk.Action, tc.wantAction)
			}
			if tc.wantContains != "" && !strings.Contains(pk.Reason+pk.Note, tc.wantContains) {
				t.Fatalf("reason/note = %q / %q, want contains %q", pk.Reason, pk.Note, tc.wantContains)
			}
		})
	}
}

func TestYumMajor(t *testing.T) {
	cases := []struct {
		distro, version, want string
		ok                    bool
	}{
		{"centos", "9", "9", true},
		{"centos", "7.9", "7", true},
		{"rocky", "9.4", "9", true},
		{"almalinux", "8.10", "8", true},
		{"alinux", "3", "8", true},
		{"alinux", "2", "7", true},
		{"opencloudos", "9", "9", true},
		{"centos", "8.5", "8", true},
		{"fedora", "40", "40", false},
	}
	for _, tc := range cases {
		got, ok := yumMajor(&DockerInstallPrecheck{Distro: tc.distro, Version: tc.version})
		if got != tc.want || ok != tc.ok {
			t.Errorf("yumMajor(%s %s) = (%q,%v), want (%q,%v)", tc.distro, tc.version, got, ok, tc.want, tc.ok)
		}
	}
}

func TestBuildStepsDebian(t *testing.T) {
	pk := &DockerInstallPrecheck{Supported: true, Action: "full", Family: "debian", Distro: "ubuntu", Codename: "jammy", Arch: "x86_64"}
	steps, err := buildSteps(pk, SourceAliyun)
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, st := range steps {
		joined.WriteString(st.Cmd + "\n")
	}
	s := joined.String()
	for _, want := range []string{
		"mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg",
		"apt-get install -y " + dockerPackages,
		"systemctl enable --now docker",
		"docker compose version",
		"/etc/apt/sources.list.d/docker.list",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("debian 步骤缺少: %s", want)
		}
	}
	// deb 源行经 b64 写文件脚本嵌入：解出还原内容断言
	re := regexp.MustCompile(`'([A-Za-z0-9+/=]{40,})'`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("未找到 b64 嵌入的仓库配置")
	}
	dec, derr := base64.StdEncoding.DecodeString(m[1])
	if derr != nil {
		t.Fatal(derr)
	}
	line := string(dec)
	for _, want := range []string{"arch=amd64 signed-by=/etc/apt/keyrings/docker.gpg", "mirrors.aliyun.com/docker-ce/linux/ubuntu jammy stable"} {
		if !strings.Contains(line, want) {
			t.Errorf("仓库配置行缺少 %s: %s", want, line)
		}
	}
}

func TestBuildStepsRhel(t *testing.T) {
	pk := &DockerInstallPrecheck{Supported: true, Action: "full", Family: "rhel", Distro: "rocky", Version: "9.4", Arch: "x86_64"}
	steps, err := buildSteps(pk, SourceTuna)
	if err != nil {
		t.Fatal(err)
	}
	var sed, all string
	for _, st := range steps {
		all += st.Cmd + "\n"
		if strings.Contains(st.Cmd, "sed -i") {
			sed = st.Cmd
		}
	}
	if !strings.Contains(all, "mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos/docker-ce.repo") {
		t.Errorf("缺少 TUNA repo 下载地址")
	}
	// 域名替换 + $releasever 显式主版本（\$ 传给 sed 为字面 $）
	if !strings.Contains(sed, `s#https://download.docker.com/linux/centos#https://mirrors.tuna.tsinghua.edu.cn/docker-ce/linux/centos#g`) {
		t.Errorf("sed 缺少官方 base 替换: %s", sed)
	}
	if !strings.Contains(sed, `s#\$releasever#9#g`) {
		t.Errorf("sed 缺少 $releasever 主版本显式化: %s", sed)
	}
}

func TestBuildStepsComposeOnly(t *testing.T) {
	pk := &DockerInstallPrecheck{Supported: true, Action: "compose-only", Family: "debian", Distro: "ubuntu", Codename: "jammy", Arch: "x86_64"}
	steps, err := buildSteps(pk, SourceOfficial)
	if err != nil {
		t.Fatal(err)
	}
	var all string
	for _, st := range steps {
		all += st.Cmd + "\n"
	}
	if !strings.Contains(all, "apt-get install -y docker-compose-plugin") {
		t.Errorf("compose-only 应只装插件")
	}
	if strings.Contains(all, dockerPackages) {
		t.Errorf("compose-only 不应安装完整包组")
	}
	if strings.Contains(all, "download.docker.com") == false {
		t.Errorf("官方源地址缺失")
	}
}

func TestParseDetect(t *testing.T) {
	out := `ID=ubuntu
VERSION_ID=22.04
CODENAME=jammy
PRETTY=Ubuntu 22.04.3 LTS
ARCH=x86_64
INIT=systemd
DBIN=/usr/bin/docker
DVER=Docker version 27.3.1, build abc
DACT=active
CVER=Docker Compose version v2.29.7`
	pk := parseDetect(out)
	if pk.Distro != "ubuntu" || pk.Codename != "jammy" || pk.Arch != "x86_64" || !pk.Systemd ||
		!pk.DockerInstalled || !pk.DockerRunning || !pk.ComposeInstalled || pk.ComposeVersion == "" {
		t.Fatalf("parseDetect 解析不完整: %+v", pk)
	}
}

func TestValidateMirrors(t *testing.T) {
	valid := []string{"https://docker.mirrors.ustc.edu.cn", "https://hub-mirror.c.163.com", "http://mirror.baidubce.com", "https://abc123.mirror.aliyuncs.com", "https://mirror.example.com:8443/v2"}
	for _, m := range valid {
		if err := validateMirrors([]string{m}); err != nil {
			t.Errorf("%s 应合法: %v", m, err)
		}
	}
	invalid := []string{"docker.mirrors.ustc.edu.cn", "https://a b.com", "https://host;rm -rf /", "file:///etc/passwd", "https://host$(id)"}
	for _, m := range invalid {
		if err := validateMirrors([]string{m}); err == nil {
			t.Errorf("%s 应拒绝", m)
		}
	}
}
