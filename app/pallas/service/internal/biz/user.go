package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/pkg/utils"
)

type User struct {
	Id         int64      `json:"id,omitempty"`
	GroupId    int64      `json:"groupId,omitempty"`
	Email      string     `json:"email,omitempty"`
	NickName   string     `json:"nickName,omitempty"`
	Password   string     `json:"password,omitempty"`
	Storage    uint64     `json:"storage,omitempty"`
	Score      int64      `json:"score,omitempty"`
	Status     UserStatus `json:"status,omitempty"`
	CreateAt   time.Time  `json:"createAt"`
	UpdateAt   time.Time  `json:"updateAt"`
	OwnerGroup *Group     `json:"ownerGroup,omitempty"`
}

type UserStatus string

// Status values.
const (
	StatusNonActivated UserStatus = "non_activated"
	StatusActive       UserStatus = "active"
	StatusBanned       UserStatus = "banned"
	StatusOveruseBaned UserStatus = "overuse_baned"
)

func (s UserStatus) String() string {
	return string(s)
}

type UserView int32

const (
	UserViewViewUnspecified UserView = 0
	UserViewBasic           UserView = 1
	UserViewWithEdgeIds     UserView = 2
)

type UserPage struct {
	Users         []*User
	NextPageToken string
}

type UserRepo interface {
	Create(ctx context.Context, user *User) (*User, error)
	Get(ctx context.Context, userId int64, userView UserView) (*User, error)
	GetByEmail(ctx context.Context, email string, userView UserView) (*User, error)
	Update(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, userId int64, email string) error
	List(ctx context.Context, pageSize int, pageToken string, userView UserView) (*UserPage, error)
	BatchCreate(ctx context.Context, users []*User) ([]*User, error)
}

type UserUsecase struct {
	repo UserRepo

	log *log.Helper
}

func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *UserUsecase) Signup(ctx context.Context, email, password string) (*v1.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, v1.ErrorGeneratePasswordError("generate password hash error: %s", err)
	}
	u := &User{
		Email:    email,
		NickName: strings.Split(email, "@")[0],
		Password: string(hashedPassword),
		Storage:  1 * utils.GibiByte,
		Score:    0,
		Status:   StatusActive,
	}
	res, err := uc.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	res.Password = ""
	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) Signin(ctx context.Context, email, password string) (*v1.User, error) {
	res, err := uc.repo.GetByEmail(ctx, email, UserViewViewUnspecified)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(res.Password), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return nil, v1.ErrorPasswordMismatch("password mismatch")
	}

	res.Password = ""
	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) GetUser(ctx context.Context, userId int64) (*v1.User, error) {
	res, err := uc.repo.Get(ctx, userId, UserViewWithEdgeIds)
	if err != nil {
		return nil, err
	}

	res.Password = ""
	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, user *User) (*v1.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	if err != nil {
		return nil, v1.ErrorGeneratePasswordError("generate password hash error: %s", err)
	}

	user.Password = string(hashedPassword)
	res, err := uc.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	res.Password = ""
	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, userId int64, email string) error {
	if err := uc.repo.Delete(ctx, userId, email); err != nil {
		return err
	}
	return nil
}

func (uc *UserUsecase) ListUsers(ctx context.Context, pageSize int, pageToken string, view UserView) ([]*v1.User, string, error) {
	page, err := uc.repo.List(ctx, pageSize, pageToken, view)
	if err != nil {
		return nil, "", err
	}

	for _, u := range page.Users {
		u.Password = ""
	}
	protoUsers, err := ToProtoUserList(page.Users)
	if err != nil {
		return nil, "", err
	}

	return protoUsers, page.NextPageToken, nil
}

func toUserStatus(p v1.User_Status) UserStatus {
	if v, ok := v1.User_Status_name[int32(p)]; ok {
		val := map[string]string{
			"NON_ACTIVATED": "non_activated",
			"ACTIVE":        "active",
			"BANNED":        "banned",
			"OVERUSE_BANED": "overuse_baned",
		}[v]
		return UserStatus(val)
	}
	return ""
}

func toProtoUserStatus(u UserStatus) v1.User_Status {
	if v, ok := v1.User_Status_value[strings.ToUpper(string(u))]; ok {
		return v1.User_Status(v)
	}
	return v1.User_Status(0)
}

func ToUser(p *v1.User) (*User, error) {
	u := &User{}
	u.Id = p.GetId()
	u.Email = p.GetEmail()
	u.NickName = p.GetNickName()
	u.Password = p.GetPassword()
	u.Storage = p.GetStorage()
	u.Score = p.GetScore()
	u.Status = toUserStatus(p.GetStatus())
	u.CreateAt = p.GetCreatedAt().AsTime()
	u.UpdateAt = p.GetUpdatedAt().AsTime()
	if p.OwnerGroup != nil {
		u.OwnerGroup = &Group{
			Id:   p.OwnerGroup.Id,
			Name: p.OwnerGroup.Name,
		}
	}
	return u, nil
}

func ToUserList(p []*v1.User) ([]*User, error) {
	userList := make([]*User, len(p))
	for i, pbEntity := range p {
		user, err := ToUser(pbEntity)
		if err != nil {
			return nil, errors.New("convert to userList error")
		}
		userList[i] = user
	}
	return userList, nil
}

func ToProtoUser(u *User) (*v1.User, error) {
	p := &v1.User{}
	p.Id = u.Id
	p.GroupId = u.GroupId
	p.Email = u.Email
	p.NickName = u.NickName
	p.Password = u.Password
	p.Storage = u.Storage
	p.Score = u.Score
	p.Status = toProtoUserStatus(u.Status)
	p.CreatedAt = timestamppb.New(u.CreateAt)
	p.UpdatedAt = timestamppb.New(u.UpdateAt)
	if u.OwnerGroup != nil {
		p.OwnerGroup = &v1.Group{
			Id:   u.OwnerGroup.Id,
			Name: u.OwnerGroup.Name,
		}
	}
	return p, nil
}

func ToProtoUserList(u []*User) ([]*v1.User, error) {
	pbList := make([]*v1.User, len(u))
	for i, userEntity := range u {
		pbUser, err := ToProtoUser(userEntity)
		if err != nil {
			return nil, errors.New("convert to protoUserList error")
		}
		pbList[i] = pbUser
	}
	return pbList, nil
}
