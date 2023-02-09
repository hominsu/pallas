package data

import "github.com/hominsu/pallas/app/pallas/service/internal/biz"

var defaultSettings = []struct {
	n string
	v string
	t biz.SettingType
}{
	{n: string(biz.RegisterEnable), v: "true", t: biz.TypeRegister},
	{n: string(biz.RegisterDefaultGroup), v: "Anonymous", t: biz.TypeRegister},
	{n: string(biz.RegisterMailActive), v: "false", t: biz.TypeRegister},
	{n: string(biz.RegisterMailFilter), v: "off", t: biz.TypeRegister},
	{n: string(biz.RegisterMailFilterList), v: "126.com,163.com," +
		"gmail.com,outlook.com,qq.com,foxmail.com,yeah.net,sohu.com,sohu.cn," +
		"139.com,wo.cn,189.cn,hotmail.com,live.com,live.cn", t: biz.TypeRegister},
}
