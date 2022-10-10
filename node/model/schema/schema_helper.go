package schema_helper

const (
	SAO_LINK_PREFIX = "sao://"
)

type SchemaHelper struct {
	// CacheSvc     *cache.CacheSvc
	//CommitSvc *commit.CommitSvc
}

func GenerateResourceId(link string) string {
	return ""
}

func GenerateResourceLink(link string) string {
	return SAO_LINK_PREFIX
}

func GetContent(link string) (interface{}, error) {
	return nil, nil
}
