package data

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/srp"
)

func newTestDataConf() []*conf.Data {
	rd := &conf.Data_Redis{
		Addr:         "127.0.0.1:6379",
		Db:           1,
		ReadTimeout:  durationpb.New(time.Millisecond * 200),
		WriteTimeout: durationpb.New(time.Millisecond * 200),
	}
	cc := &conf.Data_Cache{
		LfuSize: 10,
		Ttl:     durationpb.New(time.Second * 1),
	}

	c := []*conf.Data{
		{
			Database: &conf.Data_Database{
				Driver: "mysql",
				Source: "root:dangerous@tcp(127.0.0.1:3306)/pallas?charset=utf8mb4&parseTime=True&loc=Local",
			},
			Redis: rd,
			Cache: cc,
		},
		{
			Database: &conf.Data_Database{
				Driver: "sqlite3",
				Source: "file:ent?mode=memory&cache=shared&_fk=1",
			},
			Redis: rd,
			Cache: cc,
		},
	}
	return c
}

func newTestData(t *testing.T, c *conf.Data) (*Data, func()) {
	logger := log.With(log.NewStdLogger(io.Discard))

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	redisCache := NewRedisCache(redisCmd, c)
	Migration(entClient, params, logger)

	d, cleanup, err := NewData(entClient, redisCmd, redisCache, c, &MigrationStatus{}, logger)
	assert.NoError(t, err)

	return d, cleanup
}

func flushTestData(t *testing.T, d *Data) {
	var err error

	_, err = d.db.Setting.Delete().Exec(context.TODO())
	assert.NoError(t, err)

	_, err = d.db.User.Delete().Exec(context.TODO())
	assert.NoError(t, err)

	_, err = d.db.Group.Delete().Exec(context.TODO())
	assert.NoError(t, err)

	err = d.rdCmd.FlushDB(context.TODO()).Err()
	assert.NoError(t, err)
}

func TestMigration(t *testing.T) {
	t.Run("Check Default Group", checkDefaultGroup)
	t.Run("Check Default User", checkDefaultUser)
	t.Run("Check Default Setting", checkDefaultSetting)
}

func checkDefaultGroup(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
	}

	defaultGroupTestSuite := []struct {
		name         string
		assertion    assert.BoolAssertionFunc
		errAssertion assert.ErrorAssertionFunc
	}{
		{name: "Admin", assertion: assert.True, errAssertion: assert.NoError},
		{name: "User", assertion: assert.True, errAssertion: assert.NoError},
		{name: "Anonymous", assertion: assert.True, errAssertion: assert.NoError},
		{name: "Error", assertion: assert.False, errAssertion: assert.NoError},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range defaultGroupTestSuite {
				t.Run(tt.name, func(t *testing.T) {
					ok, err := d.data.db.Group.Query().Where(group.NameEQ(tt.name)).Exist(context.TODO())
					tt.errAssertion(t, err)
					tt.assertion(t, ok)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func checkDefaultUser(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
	}

	defaultUserTestSuite := []struct {
		email        string
		assertion    assert.BoolAssertionFunc
		errAssertion assert.ErrorAssertionFunc
	}{
		{email: "admin@pallas.icu", assertion: assert.True, errAssertion: assert.NoError},
		{email: "error@pallas.icu", assertion: assert.False, errAssertion: assert.NoError},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range defaultUserTestSuite {
				t.Run(tt.email, func(t *testing.T) {
					ok, err := d.data.db.User.Query().Where(user.EmailEQ(tt.email)).Exist(context.TODO())
					tt.errAssertion(t, err)
					tt.assertion(t, ok)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func checkDefaultSetting(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
	}

	defaultSettingTestSuite := []struct {
		n            string
		v            string
		t            biz.SettingType
		assertion    assert.ComparisonAssertionFunc
		errAssertion assert.ErrorAssertionFunc
	}{
		{n: string(biz.RegisterEnable), v: "true", t: biz.TypeRegister,
			assertion: assert.Equal, errAssertion: assert.NoError},
		{n: string(biz.RegisterDefaultGroup), v: "Anonymous", t: biz.TypeRegister,
			assertion: assert.Equal, errAssertion: assert.NoError},
		{n: string(biz.RegisterMailActive), v: "false", t: biz.TypeRegister,
			assertion: assert.Equal, errAssertion: assert.NoError},
		{n: string(biz.RegisterMailFilter), v: "off", t: biz.TypeRegister,
			assertion: assert.Equal, errAssertion: assert.NoError},
		{n: string(biz.RegisterMailFilterList), v: "126.com,163.com," +
			"gmail.com,outlook.com,qq.com,foxmail.com,yeah.net,sohu.com,sohu.cn," +
			"139.com,wo.cn,189.cn,hotmail.com,live.com,live.cn", t: biz.TypeRegister,
			assertion: assert.Equal, errAssertion: assert.NoError},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range defaultSettingTestSuite {
				t.Run(tt.n, func(t *testing.T) {
					res, err := d.data.db.Setting.Query().Where(setting.NameEQ(tt.n)).Only(context.TODO())
					tt.errAssertion(t, err)
					s, err := toSetting(res)
					tt.errAssertion(t, err)
					tt.assertion(t, tt.n, *s.Name)
					tt.assertion(t, tt.v, *s.Value)
					tt.assertion(t, tt.t, *s.Type)
				})
			}
			flushTestData(t, d.data)
		})
	}
}
