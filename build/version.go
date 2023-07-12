package build

var CurrentCommit string

const BuildVersion = "0.1.5"

func UserVersion() string {
	return CurrentCommit
}
