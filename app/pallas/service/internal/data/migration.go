package data

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/settings"
	"github.com/hominsu/pallas/pkg/srp"
	"github.com/hominsu/pallas/pkg/utils"
)

type Default struct {
	GroupsId map[string]int64
	AdminsId map[int64]struct{}
}

func Migration(entClient *ent.Client, params *srp.Params, logger log.Logger) *Default {
	helper := log.NewHelper(log.With(logger, "module", "data/migration"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !checkMigration(ctx, entClient) {
		// create default group: Admin, User and Anonymous
		createDefaultGroup(ctx, entClient, helper)

		// create default user admin@pallas.icu
		createDefaultUser(ctx, entClient, params, helper)

		// create default settings
		createDefaultSettings(ctx, entClient, helper)

		// set migration status
		setMigration(ctx, entClient)
	}

	d := &Default{}
	getDefaultGroup(ctx, entClient, d, helper)
	getAdminUsers(ctx, entClient, d, helper)
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

func getDefaultGroup(ctx context.Context, client *ent.Client, d *Default, helper *log.Helper) {
	res, err := client.Group.Query().
		Where(group.NameIn("Admin", "User", "Anonymous")).
		All(ctx)
	if err != nil {
		helper.Fatalf("failed getting default groups")
	}

	groupList, err := toGroupList(res)
	if err != nil {
		panic(err)
	}

	d.GroupsId = make(map[string]int64)
	for _, g := range groupList {
		d.GroupsId[g.Name] = g.Id
	}
}

func getAdminUsers(ctx context.Context, client *ent.Client, d *Default, helper *log.Helper) {
	res, err := client.User.Query().WithOwnerGroup(func(query *ent.GroupQuery) {
		query.Where(group.NameEQ("Admin"))
	}).All(ctx)
	if err != nil {
		helper.Fatalf("failed getting admin users")
	}

	adminList, err := toUserList(res)
	if err != nil {
		panic(err)
	}

	d.AdminsId = make(map[int64]struct{})
	for _, ad := range adminList {
		d.AdminsId[ad.Id] = struct{}{}
	}
}

func createDefaultGroup(ctx context.Context, client *ent.Client, helper *log.Helper) {
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

	err := client.Group.CreateBulk(bulk...).Exec(ctx)
	if err != nil {
		helper.Fatalf("failed creating default groups")
	}
}

func createDefaultUser(ctx context.Context, client *ent.Client, params *srp.Params, helper *log.Helper) {
	salt := []byte(utils.RandString(20, utils.AllCharSet))
	email := "admin@pallas.icu"
	password := []byte(utils.RandString(20, utils.AllCharSet))
	verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

	res := client.Group.Query().
		Where(group.NameEQ("Admin")).
		OnlyX(ctx)

	err := client.User.Create().
		SetEmail(email).
		SetNickName("admin").
		SetSalt(salt).
		SetVerifier(verifier).
		SetStorage(1 * utils.GibiByte).
		SetScore(0).
		SetStatus(user.StatusActive).
		SetOwnerGroup(res).
		Exec(ctx)
	if err != nil {
		helper.Fatalf("failed creating default user")
	}

	helper.Infof("========= default user: %s, password: %s ==========", "admin@pallas.icu", password)
}

func createDefaultSettings(ctx context.Context, client *ent.Client, helper *log.Helper) {
	s := settings.DefaultSettings()

	var bulk []*ent.SettingCreate
	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()
	for i := 0; i < typ.NumField(); i++ {
		bulk = append(bulk, client.Setting.Create().
			SetName(typ.Field(i).Name).
			SetValue(fmt.Sprintf("%v", val.Field(i).Interface())).
			SetType(toEntSettingType(biz.SettingTypeValue[typ.Field(i).Tag.Get("type")])),
		)
	}

	err := client.Setting.CreateBulk(bulk...).Exec(ctx)
	if err != nil {
		helper.Fatalf("failed creating default settings")
	}
}
