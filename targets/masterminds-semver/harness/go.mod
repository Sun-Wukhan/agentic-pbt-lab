module pbt/semver

go 1.21

require (
	github.com/Masterminds/semver/v3 v3.4.0
	pgregory.net/rapid v1.2.0
)

// Exercise the pinned clone in ../semver, never a released copy.
replace github.com/Masterminds/semver/v3 => ../semver
