package baidu

type Place struct {
	Name     string `json:"name"`
	Uid      string `json:"uid"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Address string `json:"address"`
}

type PlaceResult struct {
	Status  int     `json:"status"`
	Message string  `json:"message"`
	Total   int     `json:"total"`
	Results []Place `json:"results"`
}

type Suggestion struct {
	Name     string `json:"name"`
	Uid      string `json:"uid"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	City     string `json:"city"`
	CityId   string `json:"cityid"`
	District string `json:"district"`
}
type SuggestionResult struct {
	Status  int          `json:"status"`
	Message string       `json:"message"`
	Results []Suggestion `json:"result"`
}

type IPAddressResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Content struct {
		AddressDetail struct {
			City     string `json:"city"`
			Province string `json:"province"`
		} `json:"address_detail"`
	} `json:"content"`
}

type CityGPSResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"result"`
}

type GPSCityResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Address struct {
			Distict  string `json:"district"`
			City     string `json:"city"`
			Province string `json:"province"`
		} `json:"addressComponent"`
	} `json:"result"`
}

type CityByPhoneResult struct {
	ErrNum  int    `json:"errNum"`
	ErrMsg  string `json:"errMsg"`
	RetData struct {
		Province string `json:"province"`
		City     string `json:"city"`
		Supplier string `json:"Supplier"`
	} `json:"retData"`
}

type CityNum struct {
	Name string `json:"name"`
	Num  int    `json:"num"`
}

type CityNumResult struct {
	Status  int       `json:"status"`
	Message string    `json:"message"`
	Total   int       `json:"total"`
	Results []CityNum `json:"results"`
}
