package rorm

type RedisMode uint

const (
	_ RedisMode = iota
	Normal
	Cluster
)

type SingleNodeDesc struct {
	URL      string
	Port     string
	DB       int
	Username string
	Password string
}

type Options struct {
	Mode       RedisMode
	AddressMap map[string]*SingleNodeDesc
	ReadOnly   bool
}

//初始化
func NewRedisOptions() *Options {
	return &Options{
		Mode:       Normal,
		ReadOnly:   false,
		AddressMap: make(map[string]*SingleNodeDesc),
	}
}

func NewDefaultOptions() *Options {
	node := &SingleNodeDesc{
		URL:  "127.0.0.1",
		Port: "6379",
		DB:   0,
	}
	return &Options{
		Mode: Normal,
		AddressMap: map[string]*SingleNodeDesc{
			"default": node,
		},
	}
}

func (option *Options) SetMode(redisMode RedisMode) *Options {
	option.Mode = redisMode
	return option
}

//AddNode 添加节点 options UserName value Password value
func (option *Options) AddNode(Alias string, URL string, Port string, DB int, options ...interface{}) *Options {

	if len(options) > 0 && len(options)%2 != 0 {
		panic("parameter options must be key value")
	}

	nodeDesc := SingleNodeDesc{
		URL:  URL,
		Port: Port,
		DB:   DB,
	}

	for i := 0; i < len(options); i = i + 2 {
		if options[i] == "UserName" {
			nodeDesc.Username = options[i].(string)
		}
		if options[i] == "Password" {
			nodeDesc.Password = options[i].(string)
		}
	}

	alias := Alias
	if Alias == "" {
		alias = RandStringBytesMaskImprSrcUnsafe(6)
	}
	if option.Mode == Normal && len(option.AddressMap) > 1 {
		panic("redis is singleNode Mode.But addressList has more than one")
	}

	option.AddressMap[alias] = &nodeDesc

	return option
}

//SetReadOnly Enables read-only commands on slave nodes	(
func (option *Options) SetReadOnly(flag bool) *Options {
	if option.Mode == Normal {
		panic("normal mode can not be read only")
	}
	option.ReadOnly = flag
	return option

}
