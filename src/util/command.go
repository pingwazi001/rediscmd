package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/modood/table"
)

var clear map[string]func()
var InputReader *bufio.Reader

//key-value表格输出
type KeyValueItem struct {
	Key   string
	Value string
}

func init() {

	InputReader = bufio.NewReader(os.Stdin) //初始化

	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

func ReadConsoleLineStr() string {
	lineStr, _ := InputReader.ReadString('\n')
	lineStr = strings.ReplaceAll(lineStr, "\n", "")
	lineStr = strings.ReplaceAll(lineStr, "\r", "")
	return lineStr
}

//功能列表选择
func FuncOptionMsg() {
	msg := []KeyValueItem{
		{"1", "加载数据库列表"},
		{"2", "模糊查询指定数据库的缓存key"},
		{"3", "查询对应key的值"},
		{"4", "删除对应key的值"},
		{"5", "设置对应key的值"},
		{"6", "重新配置redis连接信息"},
		{"7", "退出程序"},
	}
	t := table.AsciiTable(msg)
	fmt.Println(t)
}

func FuncOption() {
	fmt.Println("https://github.com/pwzos/rediscmd")
	CheckConf() //检查配置文件，如果配置文件不存在就需要初始化
	for {
		FuncOptionMsg()
		fmt.Println("请输出操作选项(回车结束输入)：")
		option := ReadConsoleLineStr()
		switch option {
		case "1":
			LoadDB_1()
		case "2":
			LoadCacheKeys_2()
		case "3":
			GetValue_3()
		case "4":
			DeleteKey_4()
		case "5":
			SetValue_5()
		case "6":
			ReInitRedisConf_6()
		case "7":
			os.Exit(1)
		}
		fmt.Println("按任意键开始下一轮操作...")
		InputReader.ReadString('\n')
		CallClear()
	}
}

//加载数据
func LoadDB_1() {
	fmt.Println("是否需要展示所有数据库信息（y/n）:")
	opt := ReadConsoleLineStr()
	var dbInfoMap map[int]int
	switch opt {
	case "y":
		dbInfoMap = LoadAllDBs(true, 0)
	case "n":
		fmt.Println("请输入您需要展示前多少个数据库信息（大于0的整数）:")
		countStr := ReadConsoleLineStr()
		count, err := strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			fmt.Println("您的输入的数量无法解析，请重来")
			return
		}
		dbInfoMap = LoadAllDBs(false, count)
	default:
		fmt.Println("输入错误")
		return
	}
	for i := 0; i < len(dbInfoMap); i++ {
		fmt.Printf("db(%d)=%d\r\n", i, dbInfoMap[i])
	}
}

//加载数据库id和缓存key
func DBIdAndKeyFromConsole(isLikeKey bool) (int, string, error) {
	fmt.Println("请输入您要查询的数据库id（从0开始）:")
	dbidStr := ReadConsoleLineStr()
	dbid, err := strconv.Atoi(dbidStr)
	if err != nil {
		return 0, "", errors.New("您输入的数据库id无法解析")
	}
	msg := "请输入模糊查询Key（*表示一个或者多个任意字符串）:"
	if !isLikeKey {
		msg = "请输入将要操作的缓存Key:"
	}
	fmt.Println(msg)
	pattern := ReadConsoleLineStr()
	if pattern == "" {
		return dbid, "", errors.New("您未输入任何缓存key")
	}
	return dbid, pattern, nil
}

//加载缓存key
func LoadCacheKeys_2() {
	dbid, pattern, err := DBIdAndKeyFromConsole(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	keys, err := SearchKeys(dbid, pattern)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, item := range keys {
		item = strings.ReplaceAll(item, " ", "")
		if item == "" {
			continue
		}
		fmt.Println(item)
	}
}

//获取指定key的值
func GetValue_3() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		fmt.Println(err)
		return
	}
	value, err := GetValue(pattern, dbid)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(value)
}

//删除模糊key的值
func DeleteKey_4() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		fmt.Println(err)
		return
	}
	keys, err := SearchKeys(dbid, pattern)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, item := range keys {
		DeleteKey(dbid, item)
	}
}

func SetValue_5() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("请输入您将要设置的值:")
	valueStr := ReadConsoleLineStr()
	SetValue(dbid, pattern, valueStr)
}

func ReInitRedisConf_6() {
	for {
		err := InitConf()
		if err != nil {
			fmt.Println(err)
			continue
		}
		break
	}

}
