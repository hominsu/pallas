package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"golang.org/x/crypto/bcrypt"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/migrate"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/redisstore"
	"github.com/hominsu/pallas/pkg/utils"

	// driver
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewEntClient,
	NewRedisCmd,
	NewRedisStore,
	NewUserRepo,
	NewGroupRepo,
	Migration,
)

type Data struct {
	db    *ent.Client
	rdCmd redis.Cmdable
	store *redisstore.RedisStore

	conf *conf.Data
	d    *Default
}

type Default struct {
	groupsId []int64
}

// NewData .
func NewData(entClient *ent.Client,
	rdCmd redis.Cmdable,
	store *redisstore.RedisStore,
	conf *conf.Data,
	d *Default,
	logger log.Logger,
) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "data"))

	data := &Data{
		db:    entClient,
		rdCmd: rdCmd,
		store: store,
		conf:  conf,
		d:     d,
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

func NewRedisStore(rdCmd redis.Cmdable, conf *conf.Secret, logger log.Logger) *redisstore.RedisStore {
	helper := log.NewHelper(log.With(logger, "module", "data/redis-store"))

	store, err := redisstore.NewRedisStore(rdCmd, []byte(conf.Session.GetSessionKey()))
	if err != nil {
		helper.Fatalf("failed creating redis-store: %v", err)
	}

	return store
}

func Migration(entClient *ent.Client, logger log.Logger) *Default {
	helper := log.NewHelper(log.With(logger, "module", "data/migration"))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var groupList []*biz.Group

	if ok, err := entClient.Group.Query().Where(group.NameEQ("Admin")).Exist(ctx); err != nil {
		helper.Fatalf("failed migration: %v", err)
	} else if !ok {
		var adminId int64
		if res, err := createDefaultGroup(ctx, entClient); err != nil {
			helper.Fatalf("failed migration in create default group: %v", err)
		} else {
			for _, entEntity := range res {
				if entEntity.Name == "Admin" {
					adminId = int64(entEntity.ID)
				}
			}
			if groupList, err = toGroupList(res); err != nil {
				helper.Fatalf("internal error: %s", err)
			}
		}
		if password, err := createDefaultUser(ctx, entClient, adminId); err != nil {
			helper.Fatalf("failed migration in create default user: %v", err)
		} else {
			helper.Infof("========= default user: %s, password: %s ==========", "admin@pallas.icu", password)
		}
	}

	d := &Default{}
	for _, group := range groupList {
		d.groupsId = append(d.groupsId, group.Id)
	}

	return d
}

func createDefaultGroup(ctx context.Context, client *ent.Client) ([]*ent.Group, error) {
	now := time.Now()

	var bulk []*ent.GroupCreate
	bulk = append(bulk,
		client.Group.Create().
			SetName("Admin").
			SetMaxStorage(1*utils.GibiByte).
			SetShareEnabled(true).
			SetSpeedLimit(0).
			SetCreatedAt(now).
			SetUpdatedAt(now),
		client.Group.Create().
			SetName("User").
			SetMaxStorage(1*utils.GibiByte).
			SetShareEnabled(true).
			SetSpeedLimit(0).
			SetCreatedAt(now).
			SetUpdatedAt(now),
		client.Group.Create().
			SetName("Anonymous").
			SetMaxStorage(0).
			SetShareEnabled(true).
			SetSpeedLimit(0).
			SetCreatedAt(now).
			SetUpdatedAt(now),
	)
	groups, err := client.Group.CreateBulk(bulk...).Save(ctx)
	return groups, err
}

func createDefaultUser(ctx context.Context, client *ent.Client, adminId int64) (string, error) {
	password := utils.GeneratePassword(20, 2, 2, 2)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", err
	}

	now := time.Now()
	err = client.User.Create().
		SetEmail("admin@pallas.icu").
		SetNickName("admin").
		SetPasswordHash(string(hashedPassword)).
		SetStorage(1 * utils.GibiByte).
		SetScore(0).
		SetStatus(user.StatusActive).
		SetOwnerGroupID(int(adminId)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Exec(ctx)
	return password, err
}
