package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/pkg/srp"
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
		Addr:         "redis:6379",
		Db:           1,
		ReadTimeout:  durationpb.New(time.Millisecond * 200),
		WriteTimeout: durationpb.New(time.Millisecond * 200),
	}

	CacheConf = &conf.Data_Cache{
		LfuSize: 10,
		Ttl:     durationpb.New(time.Minute * 1),
	}

	dd *Data
)

func FlushAll(t *testing.T) {
	if err := dd.rdCmd.FlushDB(context.TODO()).Err(); err != nil {
		t.Errorf("flush redis error: %v", err)
	}

	if _, err := dd.db.User.Delete().Exec(context.TODO()); err != nil {
		t.Errorf("flush user table error: %v", err)
	}

	if _, err := dd.db.Group.Delete().Exec(context.TODO()); err != nil {
		t.Errorf("flush user table error: %v", err)
	}

	if _, err := dd.db.Setting.Delete().Exec(context.TODO()); err != nil {
		t.Errorf("flush setting table error: %v", err)
	}
}

func TestMySQL(t *testing.T) {
	var (
		err     error
		cleanup func()
	)
	logger := log.With(log.NewStdLogger(os.Stdout))

	c := &conf.Data{
		Database: MySQLConf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	}

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	redisCache := NewRedisCache(redisCmd, c)
	Migration(entClient, params, logger)

	dd, cleanup, err = NewData(entClient, redisCmd, redisCache, c, &MigrationStatus{}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	t.Run("Check MySQL", CheckMigration)
}

func TestPostgres(t *testing.T) {
	var (
		err     error
		cleanup func()
	)
	logger := log.With(log.NewStdLogger(os.Stdout))

	c := &conf.Data{
		Database: PostgreSQLConf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	}

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	redisCache := NewRedisCache(redisCmd, c)
	Migration(entClient, params, logger)

	dd, cleanup, err = NewData(entClient, redisCmd, redisCache, c, &MigrationStatus{}, logger)
	if err != nil {
		t.Fatal(err)
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

	c := &conf.Data{
		Database: SQLite3Conf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	}

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	redisCache := NewRedisCache(redisCmd, c)
	Migration(entClient, params, logger)

	dd, cleanup, err = NewData(entClient, redisCmd, redisCache, c, &MigrationStatus{}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	t.Run("Check SQLite3", CheckMigration)
}
