package bago

import "encoding/json"

type Profile struct {
	BagItProfileInfo     ProfileInfo          `json:"BagIt-Profile-Info"`
	BagInfo              map[string]TagOption `json:"Bag-Info"`
	ManifestsRequired    []string             `json:"Manifests-Required"`
	AllowFetchTxt        bool                 `json:"Allow-Fetch.txt"`
	Serialization        string               `json:"Serialization"`
	AcceptSerialization  []string             `json:"Accept-Serialization"`
	AcceptBagItVersion   []string             `json:"Accept-BagIt-Version"`
	TagManifestsRequired []string             `json:"Tag-Manifests-Required"`
	TagFilesRequired     []string             `json:"Tag-Files-Required"`
}
type tmpProfile Profile // used for Unmarshal

type ProfileInfo struct {
	SourceOrganization     string `json:"Source-Organization"`
	ExternalDescription    string `json:"External-Description"`
	Version                string `json:"Version"`
	BagItProfileIdentifier string `json:"BagIt-Profile-Identifier"`
	ContactName            string `json:"Contact-Name"`
	ContactPhone           string `json:"Contact-Phone"`
	ContactEmail           string `json:"Contact-Email"`
}

type TagOption struct {
	Values     []string
	Required   bool
	Repeatable bool
}
type tmpTagOption TagOption // used for Unmarshal

func (tag *TagOption) UnmarshalJSON(b []byte) error {
	// set non-zero defaults, which would be zeroed if missing from json
	tmp := tmpTagOption{Repeatable: true}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*tag = TagOption(tmp)
	return nil
}

func (prof *Profile) UnmarshalJSON(b []byte) error {
	// set non-zero defaults, which would be zeroed if missing from json
	tmp := tmpProfile{AllowFetchTxt: true}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*prof = Profile(tmp)
	return nil
}
