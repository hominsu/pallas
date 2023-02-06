package data

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/singleflight"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/pkg/srp"
	"github.com/hominsu/pallas/pkg/utils"
)

var testDBConf = []*conf.Data{
	{
		Database: MySQLConf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	},
	{
		Database: PostgreSQLConf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	},
	{
		Database: SQLite3Conf,
		Redis:    RedisConf,
		Cache:    CacheConf,
	},
}

func newTestUserRepo(data *Data, logger log.Logger) *userRepo {
	return &userRepo{
		data: data,
		sg:   &singleflight.Group{},
		log:  log.NewHelper(log.With(logger, "module", "data/user")),
	}
}

func initData(c *conf.Data) (*userRepo, func(), error) {
	logger := log.With(log.NewStdLogger(os.Stdout))

	params, err := srp.GetParams(2048)
	if err != nil {
		return nil, nil, err
	}

	entClient := NewEntClient(c, logger)
	redisCmd := NewRedisCmd(c, logger)
	redisCache := NewRedisCache(redisCmd, c)
	Migration(entClient, params, logger)

	ud, cleanup, err := NewData(entClient, redisCmd, redisCache, c, &MigrationStatus{}, logger)
	if err != nil {
		return nil, nil, err
	}

	tuRepo := newTestUserRepo(ud, logger)
	return tuRepo, cleanup, nil
}

func flushAll(tuRepo *userRepo) error {
	if err := tuRepo.data.rdCmd.FlushDB(context.TODO()).Err(); err != nil {
		return err
	}

	if _, err := tuRepo.data.db.User.Delete().Exec(context.TODO()); err != nil {
		return err
	}

	if _, err := tuRepo.data.db.Group.Delete().Exec(context.TODO()); err != nil {
		return err
	}

	if _, err := tuRepo.data.db.Setting.Delete().Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}

func TestUserRepo_CreateAndGet(t *testing.T) {
	for _, dbConf := range testDBConf {
		testCreateAndGet(t, dbConf)
	}
}

func testCreateAndGet(t *testing.T, dbConf *conf.Data) {
	tuRepo, cleanup, err := initData(dbConf)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	name := utils.RandString(10, utils.UpperCharSet, utils.LowerCharSet)
	email := name + "@pallas.icu"
	salt := []byte(utils.RandString(20, utils.AllCharSet))
	password := []byte(utils.RandString(20, utils.AllCharSet))
	verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

	g, err := tuRepo.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
	assert.NoError(t, err)

	var res *biz.User
	res, err = tuRepo.Create(context.TODO(), &biz.User{
		Email:      email,
		NickName:   name,
		Salt:       salt,
		Verifier:   verifier,
		Storage:    utils.GibiByte,
		Score:      0,
		Status:     biz.StatusActive,
		OwnerGroup: &biz.Group{Id: int64(g.ID)},
	})
	if err != nil {
		t.Fatalf("create test user error: %v", err)
	}

	res, err = tuRepo.Get(context.TODO(), res.Id, biz.UserViewBasic)
	if err != nil {
		t.Fatalf("get test user by id error: %v", err)
	}
	if res.Email != email {
		t.Fatalf("email not equal")
	}

	res, err = tuRepo.Get(context.TODO(), res.Id, biz.UserViewWithEdgeIds)
	if err != nil {
		t.Fatalf("get test user by id with edge error: %v", err)
	}
	if res.Email != email {
		t.Fatalf("email not equal")
	}

	res, err = tuRepo.GetByEmail(context.TODO(), res.Email, biz.UserViewBasic)
	if err != nil {
		t.Fatalf("get test user by email error: %v", err)
	}
	if res.Email != email {
		t.Fatalf("email not equal")
	}

	res, err = tuRepo.GetByEmail(context.TODO(), res.Email, biz.UserViewWithEdgeIds)
	if err != nil {
		t.Fatalf("get test user by email with edge error: %v", err)
	}
	if res.Email != email {
		t.Fatalf("email not equal")
	}

	if err = flushAll(tuRepo); err != nil {
		t.Fatalf("clean data error: %v", err)
	}
}

func TestUserRepo_Delete(t *testing.T) {
	for _, dbConf := range testDBConf {
		testDelete(t, dbConf)
	}
}

func testDelete(t *testing.T, dbConf *conf.Data) {
	tuRepo, cleanup, err := initData(dbConf)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	name := utils.RandString(10, utils.UpperCharSet, utils.LowerCharSet)
	email := name + "@pallas.icu"
	salt := []byte(utils.RandString(20, utils.AllCharSet))
	password := []byte(utils.RandString(20, utils.AllCharSet))
	verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

	g, err := tuRepo.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
	assert.NoError(t, err)

	var res *biz.User
	res, err = tuRepo.Create(context.TODO(), &biz.User{
		Email:      email,
		NickName:   name,
		Salt:       salt,
		Verifier:   verifier,
		Storage:    utils.GibiByte,
		Score:      0,
		Status:     biz.StatusActive,
		OwnerGroup: &biz.Group{Id: int64(g.ID)},
	})
	if err != nil {
		t.Fatalf("create test user error: %v", err)
	}

	if err = tuRepo.Delete(context.TODO(), res.Id); err != nil {
		t.Fatalf("delete user error: %v", err)
	}

	if _, err = tuRepo.Get(context.TODO(), res.Id, biz.UserViewBasic); ent.IsNotFound(err) {
		t.Fatalf("deleted user but user still exist")
	}

	if err = flushAll(tuRepo); err != nil {
		t.Fatalf("clean data error: %v", err)
	}
}

func TestUserRepo_BatchCreateAndList(t *testing.T) {
	for _, dbConf := range testDBConf {
		testCreateAndList(t, dbConf)
	}
}

func testCreateAndList(t *testing.T, dbConf *conf.Data) {
	tuRepo, cleanup, err := initData(dbConf)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	params, err := srp.GetParams(2048)
	assert.NoError(t, err)

	var testUsers []*biz.User
	for i := 0; i < 100; i++ {
		name := utils.RandString(10, utils.UpperCharSet, utils.LowerCharSet)
		email := name + "@pallas.icu"
		salt := []byte(utils.RandString(20, utils.AllCharSet))
		password := []byte(utils.RandString(20, utils.AllCharSet))
		verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

		g, er := tuRepo.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
		assert.NoError(t, er)

		testUsers = append(testUsers, &biz.User{
			Email:      email,
			NickName:   name,
			Salt:       salt,
			Verifier:   verifier,
			Storage:    utils.GibiByte,
			Score:      0,
			Status:     biz.StatusActive,
			OwnerGroup: &biz.Group{Id: int64(g.ID)},
		})
	}

	var res []*biz.User
	res, err = tuRepo.BatchCreate(context.TODO(), testUsers)
	if err != nil {
		t.Fatal(err)
	}

	resIndex := make(map[string]struct{})
	for _, u := range res {
		resIndex[u.Email] = struct{}{}
	}

	for _, u := range testUsers {
		if _, ok := resIndex[u.Email]; !ok {
			t.Fatalf("test user: %s not found", u.Email)
		}
	}

	type testList struct {
		pageSize int
	}

	testLists := []testList{
		{pageSize: 1},
		{pageSize: 2},
		{pageSize: 5},
		{pageSize: 10},
		{pageSize: 100},
		{pageSize: 1000},
	}

	for _, tl := range testLists {
		var users []*biz.User
		pageToken := ""
		for {
			userPage, er := tuRepo.List(context.TODO(), tl.pageSize, pageToken, biz.UserViewWithEdgeIds)
			if er != nil {
				t.Fatal(er)
			}
			users = append(users, userPage.Users...)
			if userPage.NextPageToken == "" {
				break
			}
			pageToken = userPage.NextPageToken
		}
		index := make(map[string]struct{})
		for _, u := range users {
			index[u.Email] = struct{}{}
		}
		for _, u := range testUsers {
			if _, ok := index[u.Email]; !ok && u.Email != "admin@pallas.icu" {
				t.Fatalf("test user: %s not found", u.Email)
			}
		}
	}

	if err = flushAll(tuRepo); err != nil {
		t.Fatalf("clean data error: %v", err)
	}
}
