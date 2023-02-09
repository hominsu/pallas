package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/pkg/srp"
	"github.com/hominsu/pallas/pkg/utils"
)

type User struct {
	Id         int64      `json:"id,omitempty"`
	GroupId    int64      `json:"groupId,omitempty"`
	Email      string     `json:"email,omitempty"`
	NickName   string     `json:"nickName,omitempty"`
	Salt       []byte     `json:"salt,omitempty"`
	Verifier   []byte     `json:"verifier,omitempty"`
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
	Delete(ctx context.Context, userId int64) error
	List(ctx context.Context, pageSize int, pageToken string, userView UserView) (*UserPage, error)
	BatchCreate(ctx context.Context, users []*User) ([]*User, error)

	IsAdminUser(ctx context.Context, userId int64) (bool, error)

	CacheSRPServer(ctx context.Context, email string, server *srp.Server) error
	GetSRPServer(ctx context.Context, email string) (*srp.Server, error)
}

type UserUsecase struct {
	ur     UserRepo
	gr     GroupRepo
	sr     SettingRepo
	params *srp.Params
	log    *log.Helper
}

func NewUserUsecase(ur UserRepo, gr GroupRepo, sr SettingRepo, params *srp.Params, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		ur:     ur,
		gr:     gr,
		sr:     sr,
		params: params,
		log:    log.NewHelper(logger),
	}
}

func (uc *UserUsecase) Signup(ctx context.Context, email string, salt, verifier []byte) (*v1.User, error) {
	options, err := uc.sr.ListByType(ctx, TypeRegister)
	if err != nil {
		return nil, err
	}

	// email filter
	if *options[RegisterMailFilter].Value != "off" {
		filterList := strings.Split(*options[RegisterMailFilterList].Value, ",")
		emailSplit := strings.Split(email, "@")
		filterStatus := utils.StringsContain(filterList, emailSplit[len(emailSplit)-1])
		eErr := v1.ErrorEmailDomainBanned("email domain is banned")
		if *options[RegisterMailFilter].Value == "blacklist" && filterStatus {
			return nil, eErr
		}
		if *options[RegisterMailFilter].Value == "whitelist" && !filterStatus {
			return nil, eErr
		}
	}

	get, err := uc.gr.GetByName(ctx, *options[RegisterDefaultGroup].Value, GroupViewBasic)
	if err != nil {
		return nil, err
	}
	activeRequire := *options[RegisterMailActive].Value == "true"
	ownerGroupId := get.Id

	u := &User{
		Email:      email,
		NickName:   strings.Split(email, "@")[0],
		Salt:       salt,
		Verifier:   verifier,
		Storage:    1 * utils.GibiByte,
		Score:      0,
		Status:     StatusActive,
		OwnerGroup: &Group{Id: ownerGroupId},
	}
	if activeRequire {
		u.Status = StatusNonActivated
	}

	var (
		targetUser             *User
		targetUserNotActivated = false
	)

	targetUser, err = uc.ur.Create(ctx, u)
	switch {
	case err != nil && v1.IsConflict(err):
		var uErr error
		targetUser, uErr = uc.ur.GetByEmail(ctx, email, UserViewBasic)
		if uErr != nil {
			return nil, err
		}
		if targetUser.Status == StatusNonActivated {
			targetUserNotActivated = true
		} else {
			return nil, v1.ErrorEmailExisted("email already in use")
		}
		fallthrough
	case err == nil:
		if activeRequire {
			// TODO: send email to active
		}
		// email already registered but no activated
		if targetUserNotActivated {
			return nil, v1.ErrorEmailNotActivated("user is not activated, resend the activation email")
		}
		protoUser, tErr := ToProtoUser(targetUser)
		if tErr != nil {
			return nil, tErr
		}
		return protoUser, nil
	default:
		return nil, err
	}
}

func (uc *UserUsecase) SigninA(ctx context.Context, email string, a []byte) ([]byte, error) {
	secret, err := srp.GenKey()
	if err != nil {
		return nil, v1.ErrorSigninOperation("failed in gen key: %v", err)
	}
	verifier, err := uc.getUserVerifier(ctx, email)
	if err != nil {
		return nil, err
	}

	server := srp.NewServer(uc.params, verifier, secret)
	if err = server.SetA(a); err != nil {
		return nil, v1.ErrorSigninOperation("failed in set a: %v", err)
	}
	b := server.ComputeB()
	if err = uc.ur.CacheSRPServer(ctx, email, server); err != nil {
		return nil, err
	}
	return b, nil
}

func (uc *UserUsecase) SigninM(ctx context.Context, email string, m1 []byte) (userid int64, k []byte, err error) {
	server, err := uc.ur.GetSRPServer(ctx, email)
	if err != nil {
		return 0, nil, err
	}

	_, err = server.CheckM1(m1)
	if err != nil {
		return 0, nil, v1.ErrorSigninOperation("password mismatch")
	}

	res, err := uc.ur.GetByEmail(ctx, email, UserViewBasic)
	if err != nil {
		return 0, nil, err
	}
	k = server.ComputeK()

	return res.Id, k, nil
}

func (uc *UserUsecase) GetUser(ctx context.Context, userId int64) (*v1.User, error) {
	res, err := uc.ur.Get(ctx, userId, UserViewWithEdgeIds)
	if err != nil {
		return nil, err
	}

	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) GetUserSalt(ctx context.Context, email string) ([]byte, error) {
	res, err := uc.ur.GetByEmail(ctx, email, UserViewBasic)
	if err != nil {
		return nil, err
	}

	return res.Salt, nil
}

func (uc *UserUsecase) getUserVerifier(ctx context.Context, email string) ([]byte, error) {
	res, err := uc.ur.GetByEmail(ctx, email, UserViewBasic)
	if err != nil {
		return nil, err
	}

	return res.Verifier, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, user *User) (*v1.User, error) {
	res, err := uc.ur.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	protoUser, err := ToProtoUser(res)
	if err != nil {
		return nil, err
	}

	return protoUser, nil
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, userId int64) error {
	if err := uc.ur.Delete(ctx, userId); err != nil {
		return err
	}
	return nil
}

func (uc *UserUsecase) ListUsers(
	ctx context.Context,
	pageSize int,
	pageToken string,
	view UserView,
) ([]*v1.User, string, error) {
	// list users
	page, err := uc.ur.List(ctx, pageSize, pageToken, view)
	if err != nil {
		return nil, "", err
	}

	protoUsers, err := ToProtoUserList(page.Users)
	if err != nil {
		return nil, "", err
	}

	return protoUsers, page.NextPageToken, nil
}

func (uc *UserUsecase) IsAdminUser(ctx context.Context, userId int64) (bool, error) {
	return uc.ur.IsAdminUser(ctx, userId)
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
	u.GroupId = p.GetGroupId()
	u.Email = p.GetEmail()
	u.NickName = p.GetNickName()
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
