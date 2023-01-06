package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
)

var (
	MySQLConf = &conf.Data_Database{
		Driver: "mysql",
		Source: "root:dangerous@tcp(mysql:3306)/pallas?charset=utf8mb4&parseTime=True&loc=Local",
	}

	PostgreSQLConf = &conf.Data_Database{
		Driver: "postgres",
		Source: "host=postgres port=5432 user=postgres dbname=pallas password=dangerous sslmode=disable",
	}

	SQLite3Conf = &conf.Data_Database{
		Driver: "sqlite3",
		Source: "file:ent?mode=memory&cache=shared&_fk=1",
	}

	RedisConf = &conf.Data_Redis{
		Addr:            "redis:6379",
		Db:              1,
		CacheExpiration: durationpb.New(time.Second * 1800),
		ReadTimeout:     durationpb.New(time.Millisecond * 200),
		WriteTimeout:    durationpb.New(time.Millisecond * 200),
	}

	data *Data
)

func FlushAll(t *testing.T) {
	if err := data.rdCmd.FlushAll(context.TODO()).Err(); err != nil {
		t.Errorf("flush redis error: %v", err)
	}

	if _, err := data.db.User.Delete().Exec(context.TODO()); err != nil {
		t.Errorf("flush user table error: %v", err)
	}

	if _, err := data.db.Group.Delete().Exec(context.TODO()); err != nil {
		t.Errorf("flush user table error: %v", err)
	}
}

func CheckDefault(t *testing.T) {
	for k := range data.d.GroupsId {
		if k != "Admin" && k != "User" && k != "Anonymous" {
			t.Fatalf("expected default group: %s", k)
		}
	}
}

func CheckMigration(t *testing.T) {
	t.Run("CheckDefault", CheckDefault)
	t.Run("FlushAll", FlushAll)
}

func TestMysql(t *testing.T) {
	var (
		err     error
		cleanup func()
	)
	logger := log.With(log.NewStdLogger(os.Stdout))
	helper := log.NewHelper(logger)

	c := &conf.Data{
		Database: MySQLConf,
		Redis:    RedisConf,
	}

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	d := Migration(entClient, logger)

	data, cleanup, err = NewData(entClient, redisCmd, c, d, logger)
	if err != nil {
		helper.Fatal(err)
	}
	defer cleanup()

	t.Run("Check Mysql", CheckMigration)
}

func TestPostgres(t *testing.T) {
	var (
		err     error
		cleanup func()
	)
	logger := log.With(log.NewStdLogger(os.Stdout))
	helper := log.NewHelper(logger)

	c := &conf.Data{
		Database: PostgreSQLConf,
		Redis:    RedisConf,
	}

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	d := Migration(entClient, logger)

	data, cleanup, err = NewData(entClient, redisCmd, c, d, logger)
	if err != nil {
		helper.Fatal(err)
	}
	defer cleanup()

	t.Run("Check Postgres", CheckMigration)
}

func TestSQLite3(t *testing.T) {
	var (
		err     error
		cleanup func()
	)
	logger := log.With(log.NewStdLogger(os.Stdout))
	helper := log.NewHelper(logger)

	c := &conf.Data{
		Database: SQLite3Conf,
		Redis:    RedisConf,
	}

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	d := Migration(entClient, logger)

	data, cleanup, err = NewData(entClient, redisCmd, c, d, logger)
	if err != nil {
		helper.Fatal(err)
	}
	defer cleanup()

	t.Run("Check SQLite3", CheckMigration)
}
