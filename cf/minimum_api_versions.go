package cf

import "github.com/blang/semver"

var (
	TcpRoutingMinimumAPIVersion, _                      = semver.Make("2.53.0")
	MultipleAppPortsMinimumAPIVersion, _                = semver.Make("2.51.0")
	UpdateServicePlanMinimumAPIVersion, _               = semver.Make("2.16.0")
	SetRolesByUsernameMinimumAPIVersion, _              = semver.Make("2.37.0")
	ListUsersInOrgOrSpaceWithoutUAAMinimumAPIVersion, _ = semver.Make("2.21.0")
	RoutePathMinimumAPIVersion, _                       = semver.Make("2.36.0")
	OrgAppInstanceLimitMinimumAPIVersion, _             = semver.Make("2.33.0")
	SpaceAppInstanceLimitMinimumAPIVersion, _           = semver.Make("2.40.0")
	NoaaMinimumAPIVersion, _                            = semver.Make("2.29.0")
)
