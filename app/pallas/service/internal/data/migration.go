package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/setting"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/srp"
	"github.com/hominsu/pallas/pkg/utils"
)

type MigrationStatus struct{}

func Migration(entClient *ent.Client, params *srp.Params, logger log.Logger) (status *MigrationStatus) {
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
	return &MigrationStatus{}
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
	bulk := make([]*ent.SettingCreate, len(defaultSettings))
	for i, ds := range defaultSettings {
		m := client.Setting.Create().
			SetName(ds.n).
			SetValue(ds.v).
			SetType(toEntSettingType(ds.t))
		bulk[i] = m
	}

	err := client.Setting.CreateBulk(bulk...).Exec(ctx)
	if err != nil {
		helper.Fatalf("failed creating default settings")
	}
}
