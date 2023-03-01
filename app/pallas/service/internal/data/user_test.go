package data

import (
	"context"
	"encoding/hex"
	"io"
	"regexp"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/data/ent/group"
	"github.com/hominsu/pallas/pkg/srp"
	"github.com/hominsu/pallas/pkg/utils"
)

func TestUserRepo_Create(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		user       biz.User
		ownerGroup string
		assertion  assert.ErrorAssertionFunc
	}{
		{user: biz.User{NickName: "test-1"}, ownerGroup: "Admin", assertion: assert.NoError},
		{user: biz.User{NickName: "test-1"}, ownerGroup: "Admin", assertion: assert.Error},
		{user: biz.User{NickName: "test-2"}, ownerGroup: "User", assertion: assert.NoError},
		{user: biz.User{NickName: "test-2"}, ownerGroup: "User", assertion: assert.Error},
		{user: biz.User{NickName: "test-3"}, ownerGroup: "Anonymous", assertion: assert.NoError},
		{user: biz.User{NickName: "test-3"}, ownerGroup: "Anonymous", assertion: assert.Error},
		{user: biz.User{NickName: "admin"}, ownerGroup: "Admin", assertion: assert.Error},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.user.NickName, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					target, err := d.data.db.Group.Query().Where(group.NameEQ(tt.ownerGroup)).Only(context.TODO())
					assert.NoError(t, err)

					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(tt.user.Email), password)

					tt.user.Status = biz.StatusActive
					tt.user.Email = tt.user.NickName + "@pallas.icu"
					tt.user.OwnerGroup = &biz.Group{Id: int64(target.ID)}
					tt.user.Salt = salt
					tt.user.Verifier = verifier

					_, err = d.repo.Create(context.TODO(), &tt.user)
					tt.assertion(t, err)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_Get(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		user       biz.User
		ownerGroup string
		assertion  assert.ComparisonAssertionFunc
	}{
		{user: biz.User{NickName: "test-1"}, ownerGroup: "Admin", assertion: assert.Equal},
		{user: biz.User{NickName: "test-2"}, ownerGroup: "User", assertion: assert.Equal},
		{user: biz.User{NickName: "test-3"}, ownerGroup: "Anonymous", assertion: assert.Equal},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.user.NickName, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ(tt.ownerGroup)).Only(context.TODO())
					assert.NoError(t, err)

					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(tt.user.Email), password)

					tt.user.Status = biz.StatusActive
					tt.user.Email = tt.user.NickName + "@pallas.icu"
					tt.user.OwnerGroup = &biz.Group{Id: int64(targetGroup.ID)}
					tt.user.Salt = salt
					tt.user.Verifier = verifier

					res, err := d.repo.Create(context.TODO(), &tt.user)
					assert.NoError(t, err)

					targetBasic, err := d.repo.Get(context.TODO(), res.Id, biz.UserViewBasic)
					assert.NoError(t, err)

					targetEdge, err := d.repo.Get(context.TODO(), res.Id, biz.UserViewWithEdgeIds)
					assert.NoError(t, err)

					comparison := func(expected, actual biz.User) {
						tt.assertion(t, expected.NickName, actual.NickName)
						tt.assertion(t, expected.Email, actual.Email)
						tt.assertion(t, expected.Salt, actual.Salt)
						tt.assertion(t, expected.Verifier, actual.Verifier)
					}

					comparison(*targetBasic, *targetEdge)
					comparison(tt.user, *targetBasic)
					comparison(tt.user, *targetEdge)
					tt.assertion(t, tt.user.OwnerGroup.Id, targetEdge.OwnerGroup.Id)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		name      string
		create    bool
		assertion assert.ErrorAssertionFunc
	}{
		{name: "test-1", create: true, assertion: assert.NoError},
		{name: "test-2", create: false, assertion: assert.Error},
		{name: "admin", create: false, assertion: assert.NoError},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.name, func(t *testing.T) {
					if tt.create {
						params, err := srp.GetParams(2048)
						assert.NoError(t, err)

						targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
						assert.NoError(t, err)

						email := tt.name + "@pallas.icu"
						salt := []byte(utils.RandString(20, utils.AllCharSet))
						password := []byte(utils.RandString(20, utils.AllCharSet))
						verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

						user := &biz.User{
							Email:      email,
							NickName:   tt.name,
							Salt:       salt,
							Verifier:   verifier,
							Status:     biz.StatusActive,
							OwnerGroup: &biz.Group{Id: int64(targetGroup.ID)},
						}

						_, err = d.repo.Create(context.TODO(), user)
						assert.NoError(t, err)
					}

					_, err := d.repo.GetByEmail(context.TODO(), tt.name+"@pallas.icu", biz.UserViewBasic)
					tt.assertion(t, err)

					_, err = d.repo.GetByEmail(context.TODO(), tt.name+"@pallas.icu", biz.UserViewWithEdgeIds)
					tt.assertion(t, err)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_Update(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		name        string
		group       string
		updateGroup string
		assertion   assert.ComparisonAssertionFunc
	}{
		{name: "test-1", group: "Anonymous", updateGroup: "User", assertion: assert.Equal},
		{name: "test-2", group: "Anonymous", updateGroup: "Anonymous", assertion: assert.Equal},
		{name: "test-3", group: "Anonymous", updateGroup: "Admin", assertion: assert.Equal},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.name, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ(tt.group)).Only(context.TODO())
					assert.NoError(t, err)

					email := tt.name + "@pallas.icu"
					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

					user := &biz.User{
						Email:      email,
						NickName:   tt.name,
						Salt:       salt,
						Verifier:   verifier,
						Status:     biz.StatusActive,
						OwnerGroup: &biz.Group{Id: int64(targetGroup.ID)},
					}

					res, err := d.repo.Create(context.TODO(), user)
					assert.NoError(t, err)

					updateGroup, err := d.data.db.Group.Query().Where(group.NameEQ(tt.updateGroup)).Only(context.TODO())
					assert.NoError(t, err)

					res.OwnerGroup = &biz.Group{Id: int64(updateGroup.ID)}
					_, err = d.repo.Update(context.TODO(), res)
					assert.NoError(t, err)

					target, err := d.repo.GetByEmail(context.TODO(), tt.name+"@pallas.icu", biz.UserViewWithEdgeIds)
					tt.assertion(t, res.OwnerGroup.Id, target.OwnerGroup.Id)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_Delete(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		name      string
		delete    bool
		assertion assert.ErrorAssertionFunc
	}{
		{name: "test-1", delete: true, assertion: assert.Error},
		{name: "test-2", delete: false, assertion: assert.NoError},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.name, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
					assert.NoError(t, err)

					email := tt.name + "@pallas.icu"
					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

					user := &biz.User{
						Email:      email,
						NickName:   tt.name,
						Salt:       salt,
						Verifier:   verifier,
						Status:     biz.StatusActive,
						OwnerGroup: &biz.Group{Id: int64(targetGroup.ID)},
					}

					res, err := d.repo.Create(context.TODO(), user)
					assert.NoError(t, err)

					if tt.delete {
						err = d.repo.Delete(context.TODO(), res.Id)
						assert.NoError(t, err)
					}

					_, err = d.repo.GetByEmail(context.TODO(), tt.name+"@pallas.icu", biz.UserViewBasic)
					tt.assertion(t, err)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_List(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		name      string
		assertion assert.ComparisonAssertionFunc
	}{
		{name: "test-1", assertion: assert.Equal},
		{name: "test-2", assertion: assert.Equal},
		{name: "test-3", assertion: assert.Equal},
		{name: "test-4", assertion: assert.Equal},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			nextPageToken := ""
			for i, tt := range userTestSuite {
				t.Run(tt.name, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ("Anonymous")).Only(context.TODO())
					assert.NoError(t, err)

					email := tt.name + "@pallas.icu"
					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(email), password)

					user := &biz.User{
						Email:      email,
						NickName:   tt.name,
						Salt:       salt,
						Verifier:   verifier,
						Status:     biz.StatusActive,
						OwnerGroup: &biz.Group{Id: int64(targetGroup.ID)},
					}

					_, err = d.repo.Create(context.TODO(), user)
					assert.NoError(t, err)

					res1, err := d.repo.List(context.TODO(), 1, nextPageToken, biz.UserViewBasic)
					res2, err := d.repo.List(context.TODO(), 1, nextPageToken, biz.UserViewWithEdgeIds)
					assert.Equal(t, res1.NextPageToken, res2.NextPageToken)

					nextPageToken = res1.NextPageToken
					assert.NoError(t, err)
					if i > 0 {
						tt.assertion(t, userTestSuite[i-1].name, res1.Users[0].NickName)
						tt.assertion(t, userTestSuite[i-1].name, res2.Users[0].NickName)
					} else {
						tt.assertion(t, "admin", res1.Users[0].NickName)
						tt.assertion(t, "admin", res2.Users[0].NickName)
					}
				})
			}
			flushTestData(t, d.data)
		})
	}
}

/*
func TestUserRepo_BatchCreate(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		users []struct {
			user       biz.User
			ownerGroup string
		}
		assertion assert.ErrorAssertionFunc
	}{
		{
			users: []struct {
				user       biz.User
				ownerGroup string
			}{
				{user: biz.User{NickName: "test-1"}, ownerGroup: "Admin"},
				{user: biz.User{NickName: "test-2"}, ownerGroup: "User"},
				{user: biz.User{NickName: "test-3"}, ownerGroup: "Anonymous"},
			},
			assertion: assert.NoError,
		},
		{
			users: []struct {
				user       biz.User
				ownerGroup string
			}{
				{user: biz.User{NickName: "test-4"}, ownerGroup: "Anonymous"},
				{user: biz.User{NickName: "test-5"}, ownerGroup: "User"},
				{user: biz.User{NickName: "admin"}, ownerGroup: "Admin"},
			},
			assertion: assert.Error,
		},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for i, tt := range userTestSuite {
				t.Run(fmt.Sprintf("batch-%d", i), func(t *testing.T) {
					var users []*biz.User
					for _, u := range tt.users {
						params, err := srp.GetParams(2048)
						assert.NoError(t, err)

						target, err := d.data.db.Group.Query().Where(group.NameEQ(u.ownerGroup)).Only(context.TODO())
						assert.NoError(t, err)

						salt := []byte(utils.RandString(20, utils.AllCharSet))
						password := []byte(utils.RandString(20, utils.AllCharSet))
						verifier := srp.ComputeVerifier(params, salt, []byte(u.user.Email), password)

						u.user.Status = biz.StatusActive
						u.user.Email = u.user.NickName + "@pallas.icu"
						u.user.OwnerGroup = &biz.Group{Id: int64(target.ID)}
						u.user.Salt = salt
						u.user.Verifier = verifier

						users = append(users, &u.user)
					}

					_, err := d.repo.BatchCreate(context.TODO(), users)
					tt.assertion(t, err)
				})
			}
			flushTestData(t, d.data)
		})
	}
}
*/

func TestUserRepo_IsAdminUser(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	userTestSuite := []struct {
		user       biz.User
		ownerGroup string
		assertion  assert.BoolAssertionFunc
	}{
		{user: biz.User{NickName: "test-1"}, ownerGroup: "Admin", assertion: assert.True},
		{user: biz.User{NickName: "test-2"}, ownerGroup: "User", assertion: assert.False},
		{user: biz.User{NickName: "test-3"}, ownerGroup: "Anonymous", assertion: assert.False},
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()
			for _, tt := range userTestSuite {
				t.Run(tt.user.NickName, func(t *testing.T) {
					params, err := srp.GetParams(2048)
					assert.NoError(t, err)

					targetGroup, err := d.data.db.Group.Query().Where(group.NameEQ(tt.ownerGroup)).Only(context.TODO())
					assert.NoError(t, err)

					salt := []byte(utils.RandString(20, utils.AllCharSet))
					password := []byte(utils.RandString(20, utils.AllCharSet))
					verifier := srp.ComputeVerifier(params, salt, []byte(tt.user.Email), password)

					tt.user.Status = biz.StatusActive
					tt.user.Email = tt.user.NickName + "@pallas.icu"
					tt.user.OwnerGroup = &biz.Group{Id: int64(targetGroup.ID)}
					tt.user.Salt = salt
					tt.user.Verifier = verifier

					target, err := d.repo.Create(context.TODO(), &tt.user)
					assert.NoError(t, err)

					res, err := d.repo.IsAdminUser(context.TODO(), target.Id)
					assert.NoError(t, err)
					tt.assertion(t, res)
				})
			}
			flushTestData(t, d.data)
		})
	}
}

func TestUserRepo_SRPServer(t *testing.T) {
	cs := newTestDataConf()
	ds := make([]struct {
		data    *Data
		cleanup func()
		repo    biz.UserRepo
	}, len(cs))
	for i, c := range cs {
		ds[i].data, ds[i].cleanup = newTestData(t, c)
		ds[i].repo = NewUserRepo(ds[i].data, log.With(log.NewStdLogger(io.Discard)))
	}

	for _, d := range ds {
		t.Run(d.data.conf.Database.Driver, func(t *testing.T) {
			defer d.cleanup()

			params, _ := srp.GetParams(1024)
			I := []byte("alice")
			P := []byte("password123")
			s := bytesFromHexString("beb25379d1a8581eb5a727673a2441ee")
			b := bytesFromHexString("e487cb59d31ac550471e81f00f6928e01dda08e974a004f49e61f5d105284d20")

			verifier := srp.ComputeVerifier(params, s, I, P)

			expected := map[string][]byte{
				"B": bytesFromHexString(`
			bd0c6151 2c692c0c b6d041fa 01bb152d 4916a1e7 7af46ae1 05393011
			baf38964 dc46a067 0dd125b9 5a981652 236f99d9 b681cbf8 7837ec99
			6c6da044 53728610 d0c6ddb5 8b318885 d7d82c7f 8deb75ce 7bd4fbaa
			37089e6f 9c6059f3 88838e7a 00030b33 1eb76840 910440b1 b27aaeae
			eb4012b7 d7665238 a8e3fb00 4b117b58`),
			}

			// B
			err := d.repo.CacheSRPServer(context.TODO(), "admin@pallas.icu", srp.NewServer(params, verifier, b))
			assert.NoError(t, err)

			server, err := d.repo.GetSRPServer(context.TODO(), "admin@pallas.icu")
			assert.NoError(t, err)
			assert.Equal(t, expected["B"], server.ComputeB(), "B should match")

			flushTestData(t, d.data)
		})
	}
}

func bytesFromHexString(s string) []byte {
	re, _ := regexp.Compile("[^0-9a-fA-F]")
	h := re.ReplaceAll([]byte(s), []byte(""))
	b, _ := hex.DecodeString(string(h))
	return b
}
