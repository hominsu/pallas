package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

type Group struct {
	Id          int64     `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	MaxStorage  uint64    `json:"maxStorage,omitempty"`
	ShareEnable bool      `json:"shareEnable,omitempty"`
	SpeedLimit  int64     `json:"speedLimit,omitempty"`
	CreateAt    time.Time `json:"createAt"`
	UpdateAt    time.Time `json:"updateAt"`
	Users       []*User   `json:"users,omitempty"`
}

type GroupView int32

const (
	GroupViewViewUnspecified GroupView = 0
	GroupViewBasic           GroupView = 1
	GroupViewWithEdgeIds     GroupView = 2
)

type GroupPage struct {
	Groups        []*Group
	NextPageToken string
}

type GroupRepo interface {
	Create(ctx context.Context, group *Group) (*Group, error)
	Get(ctx context.Context, groupId int64, groupView GroupView) (*Group, error)
	Update(ctx context.Context, group *Group) (*Group, error)
	Delete(ctx context.Context, groupId int64) error
	List(ctx context.Context, pageSize int, pageToken string, groupView GroupView) (*GroupPage, error)
	BatchCreate(ctx context.Context, groups []*Group) ([]*Group, error)
}

type GroupUsecase struct {
	repo GroupRepo

	log *log.Helper
}

func NewGroupUsecase(repo GroupRepo, logger log.Logger) *GroupUsecase {
	return &GroupUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func ToGroup(p *v1.Group) (*Group, error) {
	g := &Group{}
	g.Id = p.GetId()
	g.Name = p.GetName()
	g.MaxStorage = p.GetMaxStorage()
	g.ShareEnable = p.GetShareEnabled()
	g.SpeedLimit = p.GetSpeedLimit()
	g.CreateAt = p.GetCreatedAt().AsTime()
	g.UpdateAt = p.GetUpdatedAt().AsTime()
	for _, user := range p.Users {
		g.Users = append(g.Users, &User{
			Id:       user.GetId(),
			NickName: user.GetNickName(),
			Status:   toUserStatus(user.GetStatus()),
		})
	}
	return g, nil
}

func ToGroupList(p []*v1.Group) ([]*Group, error) {
	groupList := make([]*Group, len(p))
	for _, pbEntity := range p {
		group, err := ToGroup(pbEntity)
		if err != nil {
			return nil, errors.New("convert to groupList error")
		}
		groupList = append(groupList, group)
	}
	return groupList, nil
}

func ToProtoGroup(g *Group) (*v1.Group, error) {
	p := &v1.Group{}
	p.Id = g.Id
	p.Name = g.Name
	p.MaxStorage = g.MaxStorage
	p.ShareEnabled = g.ShareEnable
	p.SpeedLimit = g.SpeedLimit
	p.CreatedAt = timestamppb.New(g.CreateAt)
	p.UpdatedAt = timestamppb.New(g.UpdateAt)
	for _, user := range g.Users {
		p.Users = append(p.Users, &v1.User{
			Id:       user.Id,
			NickName: user.NickName,
			Status:   toProtoUserStatus(user.Status),
		})
	}
	return p, nil
}

func ToProtoGroupList(g []*Group) ([]*v1.Group, error) {
	pbList := make([]*v1.Group, len(g))
	for _, groupEntity := range g {
		pbGroup, err := ToProtoGroup(groupEntity)
		if err != nil {
			return nil, errors.New("convert to protoGroupList error")
		}
		pbList = append(pbList, pbGroup)
	}
	return pbList, nil
}
