// sshctl — YPanel 部署辅助：基于密码认证的 SSH exec / 文件推送。
// 密码经环境变量 SSHPASS 传入（避免落 argv/脚本）；主机指纹不做校验（仅限内网测试环境）。
//
// 用法：
//
//	sshctl -host 1.2.3.4 [-port 22] [-user root] exec -- <command>
//	sshctl -host 1.2.3.4 [-port 22] [-user root] put -local <file> -remote <path>
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	args := os.Args[1:]
	opts := map[string]string{"port": "22", "user": "root"}
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-host", "-port", "-user", "-pass":
			if i+1 >= len(args) {
				fatal("参数缺少值: " + args[i])
			}
			opts[args[i][1:]] = args[i+1]
			i++
		case "exec", "put":
			positional = append(positional, args[i])
			positional = append(positional, args[i+1:]...)
			i = len(args)
		default:
			usage()
		}
	}
	if len(positional) == 0 {
		usage()
	}

	host := opts["host"]
	if host == "" {
		fatal("缺少 -host")
	}
	password := opts["pass"]
	if password == "" {
		password = os.Getenv("SSHPASS")
	}
	if password == "" {
		fatal("缺少密码：-pass 或环境变量 SSHPASS")
	}

	client, err := dial(host, opts["port"], opts["user"], password)
	if err != nil {
		fatal("连接失败: " + err.Error())
	}
	defer func() { _ = client.Close() }()

	switch positional[0] {
	case "exec":
		cmd := positional[1:]
		if len(cmd) > 0 && cmd[0] == "--" {
			cmd = cmd[1:]
		}
		if len(cmd) == 0 {
			fatal("exec 需要 command")
		}
		code := run(client, strings.Join(cmd, " "), nil, os.Stdout, os.Stderr)
		os.Exit(code)
	case "put":
		if len(positional) < 3 || positional[1] != "-local" {
			fatal("put 用法: put -local <file> -remote <path>")
		}
		local, remote := positional[2], positional[4]
		if err := put(client, local, remote); err != nil {
			fatal("推送失败: " + err.Error())
		}
		fmt.Printf("已推送 %s -> %s:%s\n", local, host, remote)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "用法: sshctl -host H [-port 22] [-user root] {exec -- <cmd> | put -local <f> -remote <p>}")
	os.Exit(2)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "sshctl: "+msg)
	os.Exit(1)
}

func dial(host, port, user, password string) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(password)},
		// 测试环境接受任意主机指纹；生产应替换为 known_hosts 校验
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10_000_000_000,
	}
	return ssh.Dial("tcp", net.JoinHostPort(host, port), cfg)
}

// run 执行命令；stdin 为 nil 时关闭。返回退出码。
func run(client *ssh.Client, cmd string, stdin io.Reader, stdout, stderr io.Writer) int {
	sess, err := client.NewSession()
	if err != nil {
		fatal("创建会话失败: " + err.Error())
	}
	defer func() { _ = sess.Close() }()
	sess.Stdout = stdout
	sess.Stderr = stderr
	if stdin != nil {
		sess.Stdin = stdin
	}
	if err := sess.Run(cmd); err != nil {
		exitErr, ok := err.(*ssh.ExitError)
		if ok {
			return exitErr.ExitStatus()
		}
		fatal("执行失败: " + err.Error())
	}
	return 0
}

// put 经 `cat > remote` 流式推送文件（覆盖远端）。
func put(client *ssh.Client, local, remote string) error {
	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	remote = strings.ReplaceAll(remote, "\"", "\\\"")
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()

	sess.Stdin = f
	sess.Stdout = io.Discard
	sess.Stderr = os.Stderr
	if err := sess.Run(fmt.Sprintf("cat > \"%s\" && chmod 755 \"%s\" 2>/dev/null; true", remote, remote)); err != nil {
		return err
	}
	return nil
}
