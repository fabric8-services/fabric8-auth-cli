module github.com/fabric8-services/fabric8-auth-cli

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/armon/go-metrics v0.0.0-20180917152333-f0300d1749da // indirect
	github.com/danieljoos/wincred v1.0.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/fabric8-services/fabric8-auth-client v0.0.0-20190313133658-59d93fb7492b // indirect
	github.com/fabric8-services/fabric8-common v0.0.0-20190529115613-c08218d5adf2
	github.com/godbus/dbus v0.0.0-20181101234600-2ff6f7ffd60f // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/jinzhu/gorm v1.9.12 // indirect
	github.com/lib/pq v1.3.0 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/pascaldekloe/goe v0.1.0 // indirect
	github.com/pkg/errors v0.8.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/afero v1.2.0 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/zalando/go-keyring v0.0.0-20180221093347-6d81c293b3fb
	golang.org/x/crypto v0.0.0-20191205180655-e7c4368fe9dd
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
	gopkg.in/h2non/gock.v1 v1.0.15 // indirect
	gopkg.in/square/go-jose.v2 v2.2.1 // indirect
)

// required because fabric8-common requires some generated files for its tests, and those
// file are not on GitHub, so the import of the module fails. The workaround here is to
// use a local repo in which the test were run, so the code compiles.
// This local repo also needs go modules support
replace github.com/fabric8-services/fabric8-common => github.com/xcoulon/fabric8-common v0.0.0-20200312131248-f58c6dd3cd8c
