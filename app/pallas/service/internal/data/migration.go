package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/utils"
)

type Default struct {
	GroupsId map[string]int64
	AdminsId map[int64]struct{}
}

func Migration(entClient *ent.Client, logger log.Logger) *Default {
	helper := log.NewHelper(log.With(logger, "module", "data/migration"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !checkMigration(ctx, entClient) {
		createDefaultGroup(ctx, entClient)
		createDefaultUser(ctx, entClient, helper)
		setMigration(ctx, entClient)
	}

	d := &Default{}
	getDefaultGroup(ctx, entClient, d)
	getAdminUsers(ctx, entClient, d)
	return d
}

func checkMigration(ctx context.Context, client *ent.Client) bool {
	res, err := client.Setting.Query().Where(setting.NameEQ("migration")).Only(ctx)
	if err != nil && ent.IsNotFound(err) {
		return false
	}
	return res.Value == "true"
}

func setMigration(ctx context.Context, client *ent.Client) {
	client.Setting.Create().
		SetName("migration").
		SetValue("true").
		SetType(setting.TypeBasic).
		ExecX(ctx)
}

func getDefaultGroup(ctx context.Context, client *ent.Client, d *Default) {
	res := client.Group.Query().
		Where(group.NameIn("Admin", "User", "Anonymous")).
		AllX(ctx)
	groupList, err := toGroupList(res)
	if err != nil {
		panic(err)
	}

	d.GroupsId = make(map[string]int64)
	for _, g := range groupList {
		d.GroupsId[g.Name] = g.Id
	}
}

func getAdminUsers(ctx context.Context, client *ent.Client, d *Default) {
	res := client.User.Query().WithOwnerGroup(func(query *ent.GroupQuery) {
		query.Where(group.NameEQ("Admin"))
	}).AllX(ctx)
	adminList, err := toUserList(res)
	if err != nil {
		panic(err)
	}

	d.AdminsId = make(map[int64]struct{})
	for _, ad := range adminList {
		d.AdminsId[ad.Id] = struct{}{}
	}
}

func createDefaultGroup(ctx context.Context, client *ent.Client) {
	var bulk []*ent.GroupCreate
	bulk = append(bulk,
		client.Group.Create().
			SetName("Admin").
			SetMaxStorage(1*utils.GibiByte).
			SetShareEnabled(true).
			SetSpeedLimit(0),
		client.Group.Create().
			SetName("User").
			SetMaxStorage(1*utils.GibiByte).
			SetShareEnabled(true).
			SetSpeedLimit(0),
		client.Group.Create().
			SetName("Anonymous").
			SetMaxStorage(0).
			SetShareEnabled(true).
			SetSpeedLimit(0),
	)
	client.Group.CreateBulk(bulk...).ExecX(ctx)
}

func createDefaultUser(ctx context.Context, client *ent.Client, helper *log.Helper) {
	password := utils.RandString(20, utils.AllCharSet)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		panic(err)
	}

	res := client.Group.Query().
		Where(group.NameEQ("Admin")).
		OnlyX(ctx)

	client.User.Create().
		SetEmail("admin@pallas.icu").
		SetNickName("admin").
		SetPasswordHash(hashedPassword).
		SetStorage(1 * utils.GibiByte).
		SetScore(0).
		SetStatus(user.StatusActive).
		SetOwnerGroup(res).
		ExecX(ctx)

	helper.Infof("========= default user: %s, password: %s ==========", "admin@pallas.icu", password)
}
