module github.com/nsrip-dd/cmemprof/internal/testapp

go 1.17

require (
	github.com/benesch/cgosymbolizer v0.0.0-20190515212042-bec6fe6e597b
	github.com/nsrip-dd/cmemprof v1.2.3
	github.com/pkg/profile v1.6.0
)

require (
	github.com/google/pprof v0.0.0-20220314021825-5bba342933ea // indirect
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20220217162856-c813f11194b9 // indirect
)

replace (
	github.com/nsrip-dd/cmemprof => ../../
)
