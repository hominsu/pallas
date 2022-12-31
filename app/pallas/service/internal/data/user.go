package data

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/user"
	"github.com/hominsu/pallas/pkg/pagination"
)

var _ biz.UserRepo = (*userRepo)(nil)

type userRepo struct {
	data *Data
	log  *log.Helper
}

// NewUserRepo .
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/user")),
	}
}

func toUserStatus(e user.Status) biz.UserStatus { return biz.UserStatus(e) }

func toEntUserStatus(u biz.UserStatus) user.Status { return user.Status(u) }

func toUser(e *ent.User) (*biz.User, error) {
	u := &biz.User{}
	u.Id = int64(e.ID)
	u.Email = e.Email
	u.NickName = e.NickName
	u.Password = e.PasswordHash
	u.Storage = e.Storage
	u.Score = int64(e.Score)
	u.Status = toUserStatus(e.Status)
	u.CreateAt = e.CreatedAt
	u.UpdateAt = e.UpdatedAt
	if edg := e.Edges.OwnerGroup; edg != nil {
		u.OwnerGroup = &biz.Group{
			Id:   int64(edg.ID),
			Name: edg.Name,
		}
	}
	return u, nil
}

func toUserList(e []*ent.User) ([]*biz.User, error) {
	var userList []*biz.User
	for _, entEntity := range e {
		user, err := toUser(entEntity)
		if err != nil {
			return nil, errors.New("convert to userList error")
		}
		userList = append(userList, user)
	}
	return userList, nil
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	m, err := r.createBuilder(user)
	if err != nil {
		return nil, v1.ErrorInternalError("internal error: %s", err)
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		u, err := toUser(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return u, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Get(ctx context.Context, userId int64, userView biz.UserView) (*biz.User, error) {
	var (
		err error
		get *ent.User
	)
	id := int(userId)
	switch userView {
	case biz.UserViewViewUnspecified, biz.UserViewBasic:
		get, err = r.data.db.User.Get(ctx, id)
	case biz.UserViewWithEdgeIds:
		get, err = r.data.db.User.Query().
			Where(user.ID(id)).
			WithOwnerGroup(func(query *ent.GroupQuery) {
				query.Select(group.FieldID)
				query.Select(group.FieldName)
			}).
			Only(ctx)
	default:
		return nil, v1.ErrorInvalidArgument("invalid argument: unknown view")
	}
	switch {
	case err == nil:
		return toUser(get)
	case ent.IsNotFound(err):
		return nil, v1.ErrorNotFoundError("not found: %s", err)
	default:
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Update(ctx context.Context, user *biz.User) (*biz.User, error) {
	m := r.data.db.User.UpdateOneID(int(user.Id))
	m.SetEmail(user.Email)
	m.SetNickName(user.NickName)
	m.SetPasswordHash(user.Password)
	m.SetStorage(user.Storage)
	m.SetScore(int(user.Score))
	m.SetStatus(toEntUserStatus(user.Status))
	m.SetUpdatedAt(time.Now())
	if user.OwnerGroup != nil {
		m.SetOwnerGroupID(int(user.OwnerGroup.Id))
	}
	res, err := m.Save(ctx)
	switch {
	case err == nil:
		u, err := toUser(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return u, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) Delete(ctx context.Context, userId int64) error {
	var err error
	err = r.data.db.User.DeleteOneID(int(userId)).Exec(ctx)
	switch {
	case err == nil:
		return nil
	case ent.IsNotFound(err):
		return v1.ErrorNotFoundError("not found: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) List(ctx context.Context, pageSize int, pageToken string, userView biz.UserView) (*biz.UserPage, error) {
	var (
		err     error
		entList []*ent.User
	)
	listQuery := r.data.db.User.Query().
		Order(ent.Asc(user.FieldID)).
		Limit(pageSize + 1)
	if pageToken != "" {
		token, err := pagination.DecodePageToken(pageToken)
		if err != nil {
			return nil, v1.ErrorDecodePageTokenError("%s", err)
		}
		listQuery = listQuery.Where(user.IDGTE(token))
	}
	switch userView {
	case biz.UserViewViewUnspecified, biz.UserViewBasic:
		entList, err = listQuery.All(ctx)
	case biz.UserViewWithEdgeIds:
		entList, err = listQuery.
			WithOwnerGroup(func(query *ent.GroupQuery) {
				query.Select(group.FieldID)
				query.Select(group.FieldName)
			}).
			All(ctx)
	}
	switch {
	case err == nil:
		var nextPageToken string
		if len(entList) == pageSize+1 {
			nextPageToken, err = pagination.EncodePageToken(entList[len(entList)-1].ID)
			if err != nil {
				return nil, v1.ErrorEncodePageTokenError("%s", err)
			}
			entList = entList[:len(entList)-1]
		}
		userList, err := toUserList(entList)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return &biz.UserPage{
			Users:         userList,
			NextPageToken: nextPageToken,
		}, nil
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) BatchCreate(ctx context.Context, users []*biz.User) ([]*biz.User, error) {
	if len(users) > biz.MaxBatchCreateSize {
		return nil, v1.ErrorInvalidArgument("batch size cannot be greater than %d", biz.MaxBatchCreateSize)
	}
	bulk := make([]*ent.UserCreate, len(users))
	for i, user := range users {
		var err error
		bulk[i], err = r.createBuilder(user)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
	}
	res, err := r.data.db.User.CreateBulk(bulk...).Save(ctx)
	switch {
	case err == nil:
		userList, err := toUserList(res)
		if err != nil {
			return nil, v1.ErrorInternalError("internal error: %s", err)
		}
		return userList, nil
	case sqlgraph.IsUniqueConstraintError(err):
		return nil, v1.ErrorAlreadyExistsError("user already exists: %s", err)
	case ent.IsConstraintError(err):
		return nil, v1.ErrorInvalidArgument("invalid argument: %s", err)
	default:
		r.log.Errorf("unknown err: %v", err)
		return nil, v1.ErrorUnknownError("unknown error: %s", err)
	}
}

func (r *userRepo) createBuilder(user *biz.User) (*ent.UserCreate, error) {
	m := r.data.db.User.Create()
	m.SetEmail(user.Email)
	m.SetNickName(user.NickName)
	m.SetPasswordHash(user.Password)
	m.SetStorage(user.Storage)
	m.SetScore(int(user.Score))
	m.SetStatus(toEntUserStatus(user.Status))
	now := time.Now()
	m.SetCreatedAt(now)
	m.SetUpdatedAt(now)
	if user.OwnerGroup != nil {
		m.SetOwnerGroupID(int(user.OwnerGroup.Id))
	}
	return m, nil
}
