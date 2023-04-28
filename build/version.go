package build

var CurrentCommit string

const BuildVersion = "0.1.4"

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
