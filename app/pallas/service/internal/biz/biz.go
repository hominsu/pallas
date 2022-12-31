package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewUserUsecase, NewGroupUsecase)

const (
	// MaxPageSize is the maximum page size that can be returned by a List call. Requesting page sizes larger than
	// this value will return, at most, MaxPageSize entries.
	MaxPageSize = 1000

	// MaxBatchCreateSize is the maximum number of entries that can be created by a single BatchCreate call. Requests
	// exceeding this batch size will return an error.
	MaxBatchCreateSize = 1000
)
