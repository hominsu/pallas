package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var (
		groupList []*biz.Group
		adminList []*biz.User
	)

	if ok, err := entClient.Group.Query().Where(group.NameEQ("Admin")).Exist(ctx); err != nil {
		helper.Fatalf("failed migration: %v", err)
	} else if !ok {
		var adminId int64
		res, err := createDefaultGroup(ctx, entClient)
		if err != nil {
			helper.Fatalf("failed migration in create default group: %v", err)
		}
		for _, entEntity := range res {
			if entEntity.Name == "Admin" {
				adminId = int64(entEntity.ID)
			}
		}
		groupList, err = toGroupList(res)
		if err != nil {
			helper.Fatalf("internal error: %s", err)
		}

		password, err := createDefaultUser(ctx, entClient, adminId)
		if err != nil {
			helper.Fatalf("failed migration in create default user: %v", err)
		}
		helper.Infof("========= default user: %s, password: %s ==========", "admin@pallas.icu", password)
	} else {
		res, err := getDefaultGroup(ctx, entClient)
		if err != nil {
			helper.Fatalf("failed migration in get default group: %v", err)
		}
		groupList, err = toGroupList(res)
		if err != nil {
			helper.Fatalf("internal error: %s", err)
		}
	}

	res, err := getAdminUsers(ctx, entClient)
	if err != nil {
		helper.Fatalf("failed migration in get admin user: %v", err)
	}
	adminList, err = toUserList(res)
	if err != nil {
		helper.Fatalf("internal error: %s", err)
	}

	d := &Default{}
	d.GroupsId = make(map[string]int64)
	for _, g := range groupList {
		d.GroupsId[g.Name] = g.Id
	}
	d.AdminsId = make(map[int64]struct{})
	for _, ad := range adminList {
		d.AdminsId[ad.Id] = struct{}{}
	}

	return d
}

func checkMigration(ctx context.Context, client *ent.Client) bool {
	res := client.Setting.Query().Where(setting.NameEQ("migration")).OnlyX(ctx)
	return res.Value == "true"
}

func getDefaultGroup(ctx context.Context, client *ent.Client) ([]*ent.Group, error) {
	groups, err := client.Group.Query().
		Where(group.NameIn("Admin", "User", "Anonymous")).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func getAdminUsers(ctx context.Context, client *ent.Client) ([]*ent.User, error) {
	users, err := client.User.Query().WithOwnerGroup(func(query *ent.GroupQuery) {
		query.Where(group.NameEQ("Admin"))
	}).All(ctx)
	if err != nil {
		return nil, err
	}
	return users, nil
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
		SetPasswordHash(hashedPassword).
		SetStorage(1 * utils.GibiByte).
		SetScore(0).
		SetStatus(user.StatusActive).
		SetOwnerGroupID(int(adminId)).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Exec(ctx)
	return password, err
}
