package weather

import "time"

type WeatherResponse struct {
	Icon           []uint   `json:"icon"`            // 天气图标编号，支持多个，参考：https://www.hko.gov.hk/textonly/v2/explain/wxicon_sc.htm
	IconLink       []string `json:"icon_link"`       // 天气图标链接，支持多个
	Temperature    int      `json:"temperature"`     // 温度，单位：C（取香港天文台）
	Humidity       int      `json:"humidity"`        // 湿度，单位：percent（取香港天文台）
	ForecastPeriod string   `json:"forecast_period"` // 天气预报时段
	ForecastDesc   string   `json:"forecast_desc"`   // 天气预报内容
	//WarningMessage string   `json:"warning_message"` // 天气警告信息
}

type WeatherRhrreadResponse struct {
	Icon           []uint    `json:"icon"`
	IconUpdateTime time.Time `json:"iconUpdateTime"`
	UpdateTime     time.Time `json:"updateTime"`
	Temperature    struct {
		Data []struct {
			Place string `json:"place"`
			Value int    `json:"value"`
			Unit  string `json:"unit"`
		} `json:"data"`
		RecordTime time.Time `json:"recordTime"`
	} `json:"temperature"`
	Humidity struct {
		RecordTime time.Time `json:"recordTime"`
		Data       []struct {
			Unit  string `json:"unit"`
			Value int    `json:"value"`
			Place string `json:"place"`
		} `json:"data"`
	} `json:"humidity"`
	//WarningMessage string    `json:"warningMessage"`
	//Rainfall struct {
	//	Data []struct {
	//		Unit  string `json:"unit"`
	//		Place string `json:"place"`
	//		Max   int    `json:"max"`
	//		Main  string `json:"main"`
	//	} `json:"data"`
	//	StartTime time.Time `json:"startTime"`
	//	EndTime   time.Time `json:"endTime"`
	//} `json:"rainfall"`
	//Uvindex        struct {
	//	Data []struct {
	//		Place string `json:"place"`
	//		Value int    `json:"value"`
	//		Desc  string `json:"desc"`
	//	} `json:"data"`
	//	RecordDesc string `json:"recordDesc"`
	//} `json:"uvindex"`
	//Tcmessage                  string `json:"tcmessage"`
	//MintempFrom00To09          string `json:"mintempFrom00To09"`
	//RainfallFrom00To12         string `json:"rainfallFrom00To12"`
	//RainfallLastMonth          string `json:"rainfallLastMonth"`
	//RainfallJanuaryToLastMonth string `json:"rainfallJanuaryToLastMonth"`
}

type WeatherFlwResponse struct {
	ForecastPeriod string `json:"forecastPeriod"`
	ForecastDesc   string `json:"forecastDesc"`
	//UpdateTime     time.Time `json:"updateTime"`
	//GeneralSituation  string    `json:"generalSituation"`
	//TcInfo            string    `json:"tcInfo"`
	//FireDangerWarning string    `json:"fireDangerWarning"`
	//Outlook           string    `json:"outlook"`
}
