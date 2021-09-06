package model

//redis配置文件的模型
type RedisConf struct {
	Redis struct {
		AddRess    string
		Port       int
		Password   string
		MaxConnect int    //连接池中允许最大的连接数
		KeyPrefix  string //缓存key的前缀字符
	}
}
