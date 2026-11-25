module pbt/tailscale

go 1.27.4

require (
	go4.org/mem v0.0.0-20240501181205-ae6ca9944745
	go4.org/netipx v0.0.0-20260823151212-3075585bcbeb
	pgregory.net/rapid v1.2.0
	tailscale.com v0.0.0
)

require (
	github.com/gaissmai/bart v0.29.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20260820222146-c27c302e5fc3 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/time v0.16.0 // indirect
)

// Exercise the pinned clone in ../tailscale, never a released copy.
replace tailscale.com => ../tailscale
