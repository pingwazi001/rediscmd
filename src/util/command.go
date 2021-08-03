package util

import (
	"bufio"
	"errors"
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
		{"2", "模糊查询缓存key"},
		{"3", "查询模糊key的值"},
		{"4", "删除模糊key的值"},
		{"5", "设置精确key的值"},
		{"6", fmt.Sprintf("重新配置%s文件内容", confName)},
		{"7", "切换配置文件"},
		{"8", "触发刷新本地缓存Key集合"},
		{"9", "新增配置文件"},
		{"10", "退出程序"},
	}
	t := table.AsciiTable(msg)
	fmt.Println(t)
}

func FuncOption() {
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
	for {
		FuncOptionMsg()
		log.Println("请输出操作选项(回车结束输入):")
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
			ChangeConfName_7()
		case "8":
			ReLoadAllCacheKeys_8()
		case "9":
			AddConf_9()
		case "10":
			os.Exit(1)
		}
		log.Println("按回车开始下一轮操作...")
		InputReader.ReadString('\n')
		CallClear()
	}
}

//加载数据
func LoadDB_1() {
	log.Println("是否需要展示所有数据库信息（y/n）:")
	opt := ReadConsoleLineStr()
	var dbInfoMap map[int]int
	switch opt {
	case "y":
		log.Println("正在加载数据库信息，请稍候...")
		dbInfoMap = LoadAllDBs(true, 0)
	case "n":
		log.Println("请输入您需要展示前多少个数据库信息（大于0的整数）:")
		countStr := ReadConsoleLineStr()
		count, err := strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			log.Println("您的输入的数量无法解析，请重来")
			return
		}
		log.Println("正在加载数据库信息，请稍候...")
		dbInfoMap = LoadAllDBs(false, count)
	default:
		log.Println("输入错误")
		return
	}
	for i := 0; i < len(dbInfoMap); i++ {
		log.Printf("db(%d)=%d\r\n", i, dbInfoMap[i])
	}
}

//加载数据库id和缓存key
func DBIdAndKeyFromConsole(isLikeKey bool) (int, string, error) {
	msg := fmt.Sprintf("请输入您要操作的数据[0,%d) 和 模糊查询Key(*表示一个或者多个任意字符串)！使用一个空格分隔:", DbCount)
	if !isLikeKey {
		msg = fmt.Sprintf("请输入您要操作的数据[0,%d) 和 将要操作的缓存Key！使用一个空格分隔:", DbCount)
	}
	log.Println(msg)
	dbIdAndKeyStr := ReadConsoleLineStr()
	fEmptyStrIndex := strings.Index(dbIdAndKeyStr, " ")
	if fEmptyStrIndex <= 0 {
		return 0, "", errors.New("您输入的内容无法解析！请按照规则进行输入")
	}
	dbidStr := strings.Trim(dbIdAndKeyStr[0:fEmptyStrIndex], " ")
	dbid, err := strconv.Atoi(dbidStr)
	if err != nil {
		return 0, "", errors.New("您输入的数据库id无法解析")
	}
	pattern := strings.Trim(dbIdAndKeyStr[fEmptyStrIndex:], " ")
	if pattern == "" {
		return dbid, "", errors.New("您未输入任何缓存key")
	}
	return dbid, pattern, nil
}

//加载缓存key
func LoadCacheKeys_2() {
	dbid, pattern, err := DBIdAndKeyFromConsole(true)
	if err != nil {
		log.Println(err)
		return
	}
	keys, err := SearchKeys(dbid, pattern)
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
func GetValue_3() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		log.Println(err)
		return
	}
	keys, err := SearchKeys(dbid, pattern)
	if err != nil {
		log.Println(err)
		return
	}
	if len(keys) <= 0 {
		log.Println("未查询找到任何缓存，您可以尝试刷新本地缓存key的操作再查询")
		return
	}
	valueMsgChan := make(chan string, len(keys))
	for _, key := range keys {
		go func(itemKey string) {
			value, err := GetValue(itemKey, dbid)
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
func DeleteKey_4() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		log.Println(err)
		return
	}
	if pattern == "*" {
		log.Printf("根据您输入的模糊Key=%s此次操作将清空数据库dbid=%d中的所有缓存！请确认是否执行此操作(y/n):", pattern, dbid)
		isSure := ReadConsoleLineStr()
		if isSure == "y" {
			log.Println("正在处理，请稍候...")
			FlushDB(dbid) //清空数据库
		}
		return
	}
	keys, err := SearchKeys(dbid, pattern)
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
			DeleteKey(dbid, itemKey...) //一次性批量删除多个
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

func SetValue_5() {
	dbid, pattern, err := DBIdAndKeyFromConsole(false)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("请输入您将要设置的值:")
	valueStr := ReadConsoleLineStr()
	SetValue(dbid, pattern, valueStr)
}

//重新配置当前设置当前配置文件的内容
func ReInitRedisConf_6() {
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
func ChangeConfName_7() {
	for {
		if err := initRedisConnectInfo(); err != nil {
			log.Println(err)
			continue
		}
		break
	}
}

//触发加载缓存key
func ReLoadAllCacheKeys_8() {
	//遍历删除已加载的缓存key
	CacheKeysMap.Range(func(k, v interface{}) bool {
		CacheKeysMap.Delete(k)
		return true
	})
	//触发加载缓存key
	initLoadAllCacheKeys()
}

//添加配置文件
func AddConf_9() {
	createConfFile()
}
