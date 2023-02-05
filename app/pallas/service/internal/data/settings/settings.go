package settings

type Setting struct {
	RegisterEnable         bool   `type:"register"`
	RegisterDefaultGroup   string `type:"register"`
	RegisterMailActive     bool   `type:"register"`
	RegisterMailFilter     bool   `type:"register"`
	RegisterMailFilterList string `type:"register"`
}

func DefaultSettings() *Setting {
	s := &Setting{
		RegisterEnable:       true,
		RegisterDefaultGroup: "Anonymous",
		RegisterMailActive:   false,
		RegisterMailFilter:   false,
		RegisterMailFilterList: "126.com,163.com,gmail.com," +
			"outlook.com,qq.com,foxmail.com,yeah.net,sohu.com," +
			"sohu.cn,139.com,wo.cn,189.cn,hotmail.com,live.com,live.cn",
	}
	return s
}
