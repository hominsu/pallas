package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/cache/v9"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/migrate"
	"github.com/hominsu/pallas/pkg/sessions"
	"github.com/hominsu/pallas/pkg/srp"

	// driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewEntClient,
	NewRedisCmd,
	NewRedisCache,
	NewRedisStore,
	NewSRPParams,
	NewUserRepo,
	NewGroupRepo,
	NewSettingRepo,
	Migration,
)

type Data struct {
	db    *ent.Client
	rdCmd redis.Cmdable
	cache *cache.Cache

	conf *conf.Data
}

// NewData .
func NewData(
	entClient *ent.Client,
	rdCmd redis.Cmdable,
	cache *cache.Cache,
	conf *conf.Data,
	_ *MigrationStatus,
	logger log.Logger,
) (*Data, func(), error) {
	// NewData
	helper := log.NewHelper(log.With(logger, "module", "data"))

	data := &Data{
		db:    entClient,
		rdCmd: rdCmd,
		cache: cache,
		conf:  conf,
	}
	return data, func() {
		if err := data.db.Close(); err != nil {
			helper.Error(err)
		}
	}, nil
}

func NewEntClient(conf *conf.Data, logger log.Logger) *ent.Client {
	helper := log.NewHelper(log.With(logger, "module", "data/ent"))

	client, err := ent.Open(
		conf.Database.Driver,
		conf.Database.Source,
	)
	if err != nil {
		helper.Fatalf("failed opening connection to db: %v", err)
	}
	// Run the auto migration tool.
	if err := client.Schema.Create(
		context.Background(),
		migrate.WithForeignKeys(true),
		migrate.WithGlobalUniqueID(true),
	); err != nil {
		helper.Fatalf("failed creating schema resources: %v", err)
	}
	return client
}

func NewRedisCmd(conf *conf.Data, logger log.Logger) redis.Cmdable {
	helper := log.NewHelper(log.With(logger, "module", "data/redis"))

	client := redis.NewClient(&redis.Options{
		Addr:         conf.Redis.Addr,
		Password:     conf.Redis.Password,
		DB:           int(conf.Redis.Db),
		ReadTimeout:  conf.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: conf.Redis.WriteTimeout.AsDuration(),
		DialTimeout:  time.Second * 2,
		PoolSize:     10,
	})

	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFunc()

	err := client.Ping(timeout).Err()
	if err != nil {
		helper.Fatalf("redis connect error: %v", err)
	}
	return client
}

func NewRedisCache(rdCmd redis.Cmdable, conf *conf.Data) *cache.Cache {
	opts := &cache.Options{
		Redis: rdCmd,
	}
	if conf.Cache.LfuEnable {
		opts.LocalCache = cache.NewTinyLFU(int(conf.Cache.LfuSize), conf.Cache.Ttl.AsDuration())
	}

	return cache.New(opts)
}

func NewRedisStore(rdCmd redis.Cmdable, conf *conf.Secret, logger log.Logger) *sessions.RedisStore {
	helper := log.NewHelper(log.With(logger, "module", "data/redis-store"))

	store, err := sessions.NewRedisStore(rdCmd, []byte(conf.Session.GetSessionKey()))
	store.SetMaxAge(10 * 24 * 3600)
	if err != nil {
		helper.Fatalf("failed creating redis-store: %v", err)
	}

	return store
}

func NewSRPParams(secret *conf.Secret, logger log.Logger) *srp.Params {
	helper := log.NewHelper(log.With(logger, "module", "data/srp-params"))

	params, err := srp.GetParams(int(secret.Srp.GetSrpParams()))
	if err != nil {
		helper.Fatalf("failed init params")
	}

	return params
}
