package util

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

//清屏
func ClearConsoleScreen() {
	switch runtime.GOOS {
	case "linux", "darwin":
		{
			cmd := exec.Command("clear") //Linux example, its tested
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	case "windows":
		{
			cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
	default:
		{
			log.Printf("您的系统还不支持清屏操作，快找平娃子看看能否有办法吧 :(")
		}
	}
}

//从控制塔台读取一个值
func ReadValueFromConsole(noticeMsg string, isNum bool) (string, int) {
	stdInput := bufio.NewReader(os.Stdin)
	log.Println(noticeMsg)
	str, err := stdInput.ReadString('\n')
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	str = strings.Trim(str, " ")
	if err != nil || str == "" {
		log.Println("未获取到您输入的任何内容，请重新输入！")
		return ReadValueFromConsole(noticeMsg, isNum)
	}
	if isNum {
		num, err := strconv.Atoi(str)
		if err != nil {
			log.Println("您输入了非数字内容，请重新输入！")
			return ReadValueFromConsole(noticeMsg, isNum)
		}
		return "", num
	}
	return str, 0
}
