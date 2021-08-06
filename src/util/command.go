package util

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

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
	clear["darwin"] = func() {
		cmd := exec.Command("clear") //macos
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

//启动程序
func AppStart() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("工具发生致命错误！请通过https://github.com/pwzos/rediscmd向工具作者进行反馈")
			log.Println(err)
			log.Println("按回车退出程序...")
			InputReader.ReadString('\n')
			os.Exit(1)
		}
	}()
	log.Println("https://github.com/pwzos/rediscmd")
	FuncOptionMsg() //功能提示语
	for {
		FuncOption() //功能选择
	}
}

//清屏
func ClearScreen() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

//控制台获取一个输入，回车结束输入
func ReadConsoleLineStr() string {
	lineStr, _ := InputReader.ReadString('\n')
	lineStr = strings.ReplaceAll(lineStr, "\n", "")
	lineStr = strings.ReplaceAll(lineStr, "\r", "")
	lineStr = strings.Trim(lineStr, " ")
	return lineStr
}

//功能列表选择
func FuncOptionMsg() {
	msg := []KeyValueItem{
		{"cls", "清屏"},
		{"ldb", fmt.Sprintf("加载数据库列表 [y|n] [0~%d)", DbCount)},
		{"keys", "模糊查询缓存key [keypattern]"},
		{"get", "查询模糊key的值 [keypattern]"},
		{"del", "删除模糊key的值 [keypattern]"},
		{"set", "设置精确key的值 [key] [value]"},
		{"resetconf", fmt.Sprintf("重新配置%s文件内容", confName)},
		{"changeconf", "切换配置文件"},
		{"reloadkeys", "触发刷新本地缓存Key集合"},
		{"addconf", "新增配置文件"},
		{"changeoptdbid", fmt.Sprintf("切换当前操作数据库编号%d为其他值 [0~%d)", DbCount, DbCount)},
		{"quit", "退出程序"},
	}
	t := table.AsciiTable(msg)
	fmt.Println(t)
}

//功能选择
func FuncOption() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("工具发生致命错误！请通过https://github.com/pwzos/rediscmd向工具作者进行反馈")
			log.Println(err)
			log.Println("按任意键继续操作...")
			InputReader.ReadString('\n')
			return
		}
	}()
	log.Println("请输出操作命令(回车结束输入):")
	option := ReadConsoleLineStr()
	if option == "" {
		log.Println("输入有误，请重新输入！")
		return
	}
	cmdParams := strings.Split(option, " ")
	cmdParams = dealCMDParams(cmdParams) //处理命令行参数
	switch cmdParams[0] {
	case "cls":
		ClearScreen()
		FuncOptionMsg()
	case "ldb":
		LoadDBCMD(cmdParams)
	case "keys":
		KeysCMD(cmdParams)
	case "get":
		GetCMD(cmdParams)
	case "del":
		DelCMD(cmdParams)
	case "set":
		SetCMD(cmdParams)
	case "resetconf":
		ResetConfCMD()
	case "changeconf":
		ChangeConfCMD()
	case "reloadkeys":
		ReloadKeysCMD()
	case "addconf":
		AddConfCMD()
	case "changeoptdbid":
		ChangeOptDbIdCMD(cmdParams)
	case "quit":
		os.Exit(1)
	default:
		log.Printf("您输入的操作【%s】不支持！请重新输入", cmdParams[0])
	}
}

//处理输入的命令行参数
func dealCMDParams(cmdParams []string) []string {
	retCmdParams := []string{}
	for _, item := range cmdParams {
		item = strings.Trim(item, " ")
		if item == "" {
			continue
		}
		retCmdParams = append(retCmdParams, item)
	}
	return retCmdParams
}

//检查命令行参数个数是否符合规则
func checkCMDParamsCount(cmdParams []string, count int) bool {
	return count <= len(cmdParams)
}

//加载数据库列表信息
func LoadDBCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("加载数据库列表信息至少需要两个参数，请重新输入")
		return
	}
	opt := cmdParams[1]
	var dbInfoMap map[int]int
	switch opt {
	case "y":
		log.Println("正在加载全部数据库信息，请稍候...")
		dbInfoMap = LoadAllDBs(true, 0)
	case "n":
		if !checkCMDParamsCount(cmdParams, 3) {
			log.Println("请输入需要加载前前多少个数据库的信息")
			return
		}
		count, err := strconv.Atoi(cmdParams[2])
		if err != nil || count <= 0 {
			log.Println("您的输入的数量无法解析，请重来")
			return
		}
		log.Println("正在加载数据库信息，请稍候...")
		dbInfoMap = LoadAllDBs(false, count)
	default:
		log.Println("加载方式仅允许输入y/n")
		return
	}
	for i := 0; i < len(dbInfoMap); i++ {
		log.Printf("db(%d)=%d\r\n", i, dbInfoMap[i])
	}
}

//加载缓存key
func KeysCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的列表需要两个参数，请重新输入")
		return
	}
	keys, err := SearchKeys(cmdParams[1])
	if err != nil {
		log.Println(err)
		return
	}

	for _, item := range keys {
		item = strings.ReplaceAll(item, " ", "")
		if item == "" {
			continue
		}
		log.Println(item)
	}
}

//获取指定key的值
func GetCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的值需要两个参数，请重新输入")
		return
	}
	keys, err := SearchKeys(cmdParams[1])
	if err != nil {
		log.Println(err)
		return
	}
	if len(keys) <= 0 {
		log.Println("未查询找到任何缓存，您可以尝试刷新本地key缓存后再查询")
		return
	}
	valueMsgChan := make(chan string, len(keys))
	for _, key := range keys {
		go func(itemKey string) {
			value, err := GetValue(itemKey)
			if err != nil {
				valueMsgChan <- itemKey + "=" + err.Error()
				return
			}
			valueMsgChan <- fmt.Sprintf("%s=%s", itemKey, value)
		}(key)
	}

	//打印查询结果
	for i := 0; i < len(keys); i++ {
		log.Println(<-valueMsgChan)
		log.Println()
	}
}

//删除模糊key的值
func DelCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊批量删除缓存需要两个参数，请重新输入")
		return
	}
	pattern := cmdParams[1]
	if pattern == "*" {
		log.Printf("根据您输入的模糊Key=%s此次操作将清空数据库dbid=%d中的所有缓存！请确认是否执行此操作(y/n):", pattern, OptDBId)
		isSure := ReadConsoleLineStr()
		if isSure == "y" {
			log.Println("正在处理，请稍候...")
			FlushDB() //清空数据库
		}
		return
	}
	keys, err := SearchKeys(pattern)
	if err != nil {
		log.Println(err)
		return
	}
	allKeysCount := len(keys)
	log.Printf("根据您输入的模糊Key=%s此次将批量删除%d条缓存数据，按回车开始操作", pattern, allKeysCount)
	InputReader.ReadString('\n')
	deleteMsgChan := make(chan string, allKeysCount)
	delCount := 1000
	forCount := 0
	for {
		if delCount > len(keys) {
			delCount = len(keys)
		}
		delKeys := keys[0:delCount]
		go func(itemKey []string) {
			start := time.Now()
			DeleteKey(itemKey...) //一次性批量删除多个
			deleteMsgChan <- fmt.Sprintf("%s 删除成功，耗时%d毫秒", itemKey, time.Since(start).Milliseconds())
		}(delKeys)
		forCount++
		keys = keys[delCount:] //重新指定
		if delCount >= len(keys) {
			break
		}
	}
	for i := 0; i < forCount; i++ {
		log.Println(<-deleteMsgChan)
	}
}

func SetCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 3) {
		log.Println("给指定key设置值需要三个参数，请重新输入")
		return
	}
	key := cmdParams[1]
	value := cmdParams[2]
	SetValue(key, value)
}

//重新配置当前设置当前配置文件的内容
func ResetConfCMD() {
	confInited := false
	for {
		if !confInited {
			err := InitConf()
			if err != nil {
				log.Println(err)
				continue
			}
		}
		confInited = true
		err := initRedisConnectInfo() //因为重新配置了redis的连接信息，所以需要重新初始化redis连接信息
		if err != nil {
			log.Println(err)
			continue
		}
		break
	}
}

//切换配置文件
func ChangeConfCMD() {
	for {
		if err := initRedisConnectInfo(); err != nil {
			log.Println(err)
			continue
		}
		break
	}
}

//触发加载缓存key
func ReloadKeysCMD() {
	//遍历删除已加载的缓存key
	CacheKeysMap.Range(func(k, v interface{}) bool {
		CacheKeysMap.Delete(k)
		return true
	})
	//触发加载缓存key
	initLoadAllCacheKeys()
}

//添加配置文件
func AddConfCMD() {
	createConfFile()
}

//切换操作数据
func ChangeOptDbIdCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("切换操作数据库需要两个参数，请重新输入")
		return
	}
	dbid, err := strconv.Atoi(cmdParams[1])
	if err != nil || dbid <= 0 {
		log.Println("无法解析您输入的数据库编号")
		return
	}
	ChangeDBId(dbid)
}
