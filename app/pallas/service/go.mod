module github.com/hominsu/pallas/app/pallas/service

go 1.18

require (
	entgo.io/ent v0.11.8
	github.com/go-kratos/kratos/v2 v2.5.4
	github.com/go-redis/cache/v9 v9.0.0
	github.com/go-sql-driver/mysql v1.7.0
	github.com/google/wire v0.5.0
	github.com/gorilla/handlers v1.5.1
	github.com/hominsu/pallas v0.0.0-00010101000000-000000000000
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/redis/go-redis/v9 v9.0.2
	github.com/stretchr/testify v1.8.1
	golang.org/x/sync v0.1.0
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/securecookie v1.1.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20230216225411-c8e22ba71e44 // indirect
	google.golang.org/grpc v1.53.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hominsu/pallas => ../../../
