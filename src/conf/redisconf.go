package conf

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"rediscmd/src/model"
	"rediscmd/src/util"

	"gopkg.in/gcfg.v1"
)

var (
	redisConfName    = "conf.ini"
	redisConfAbsPath = ""
)

//设置配置文件名称
func SetRedisConfName(name string) {
	redisConfName = name
	redisConfAbsPath = ""
}

//获取配置文件名称
func RedisConfName() string {
	return redisConfName
}

//conf文件的绝对路径
func RedisConfAbsPath() string {
	if redisConfAbsPath == "" {
		execPath, err := util.ExecFilePath()
		if err != nil {
			log.Printf("可执行文件路径获取失败%s，此操作仅针对当前工作目录下的配置文件有效！", err.Error())
			redisConfAbsPath = RedisConfName()
		}
		redisConfAbsPath = filepath.Join(execPath, RedisConfName())
	}
	return redisConfAbsPath
}

//初始化配置
func InitRedisConf() error {
	address, _ := util.ReadValueFromConsole("请输入redis连接地址", false)
	_, port := util.ReadValueFromConsole("请输入redis连接端口", true)
	password, _ := util.ReadValueFromConsole("请输入redis访问密码", false)
	_, maxConnect := util.ReadValueFromConsole("请输入redis连接池中允许的最大连接数（1~100）", true)

	//删除已经存在的配置文件
	util.RemoveFile(RedisConfAbsPath()) //删除配置文件
	fmt.Println(RedisConfAbsPath())
	fileObj, err := os.Create(RedisConfAbsPath())
	if err != nil {
		util.RemoveFile(RedisConfAbsPath()) //文件创建失败
		return err
	}
	defer fileObj.Close()
	writer := bufio.NewWriter(fileObj)
	writer.WriteString("[redis]\n")

	writer.WriteString(fmt.Sprintf("AddRess=%s\n", address))
	writer.WriteString(fmt.Sprintf("Port=%d\n", port))
	writer.WriteString(fmt.Sprintf("Password=%s\n", password))
	writer.WriteString(fmt.Sprintf("MaxConnect=%d", maxConnect))
	writer.Flush()
	return nil
}

//检查配置文件是否存在
func CheckRedisConf() error {
	confFileAbsPath := RedisConfAbsPath()
	//检查配置文件是否存在
	if _, err := os.Stat(confFileAbsPath); os.IsNotExist(err) {
		return fmt.Errorf("%s配置文件读取失败，请初始化此配置信息", confFileAbsPath)
	}
	config := new(model.RedisConf)
	err := gcfg.ReadFileInto(config, confFileAbsPath)
	if err != nil {
		return fmt.Errorf("%s配置文件读取失败，请初始化此配置信息", confFileAbsPath)
	}
	if config.Redis.AddRess == "" || config.Redis.Password == "" || config.Redis.Port == 0 || config.Redis.MaxConnect < 1 || config.Redis.MaxConnect > 100 {
		return fmt.Errorf("%s配置内置内容不正确，请初始化此配置信息", confFileAbsPath)
	}
	return nil
}

//获取配置
func GetRedisConf() (*model.RedisConf, error) {
	config := new(model.RedisConf)
	err := gcfg.ReadFileInto(config, RedisConfAbsPath())
	if err != nil {
		return nil, err
	}
	return config, nil
}

//选择配置文件名称
func SeleRedisctConfFileName() {
	pwd, err := util.ExecFilePath()
	if err != nil {
		pwd, _ = os.Getwd()
	}
	confFileNames, err := filepath.Glob(filepath.Join(pwd, "conf*.ini"))
	if err != nil {
		log.Printf("加载%s下的文件出错%s，使用默认配置文件(%s)！", pwd, err.Error(), RedisConfName())
		return
	}
	if len(confFileNames) <= 0 {
		log.Printf("%s下不存在任何配置文件，使用默认配置文件(%s)！", pwd, RedisConfName())
		return
	}
	fileNames := []string{}
	for _, fileName := range confFileNames {
		_, name := filepath.Split(fileName)
		fileNames = append(fileNames, name)
	}

	//只有一个配置文件，就不用选择了，直接使用
	if len(fileNames) == 1 {
		SetRedisConfName(fileNames[0])
		log.Printf("%s下只有一个配置文件(%s)，无需选择直接使用！", pwd, RedisConfName())
		return
	}
	for index, fileName := range fileNames {
		log.Printf("%d %s\r\n", index, fileName)
	}
	_, fileNO := util.ReadValueFromConsole(fmt.Sprintf("您当前存在多个配置文件，请输入您要使用的文件编号[0~%d)", len(fileNames)), true)
	if fileNO < 0 || fileNO >= len(fileNames) {
		log.Println("您输入的文件编号不在指定范围内，请重新输入")
		SeleRedisctConfFileName()
		return //防止继续往下走
	}
	SetRedisConfName(fileNames[fileNO])
}

//创建配置文件
func CreateRedisConfFile() {
	backConfName := RedisConfName()
	log.Println("新生成的配置名称名称格式将为【conf-*.ini】，其中*是需要您输入的部分")
	log.Println("如果文件存在，将覆盖之前的文件！！！")
	log.Println("如果文件存在，将覆盖之前的文件！！！")
	log.Println("如果文件存在，将覆盖之前的文件！！！")
	confCustNameStr, _ := util.ReadValueFromConsole("请输入新增配置文件的自定义部分（不能为空或者*）", false)
	if confCustNameStr == "*" {
		log.Println("您输入的配置文件自定义部分格式不正确")
		return
	}
	SetRedisConfName(fmt.Sprintf("conf-%s.ini", confCustNameStr))
	InitRedisConf()                //初始化配置文件
	SetRedisConfName(backConfName) //换回目前使用的配置文件
}
