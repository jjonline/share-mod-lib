package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jjonline/share-mod-lib/guzzle"
)

const (
	// HKWeatherApi 文档：https://data.weather.gov.hk/weatherAPI/doc/HKO_Open_Data_API_Documentation_sc.pdf
	HKWeatherApi = "https://data.weather.gov.hk/weatherAPI/opendata/weather.php"

	// HKWeatherIconLink 图标链接，https://www.hko.gov.hk/textonly/v2/explain/wxicon_sc.htm
	HKWeatherIconLink = "https://www.hko.gov.hk/images/HKOWxIconOutline/pic%d.png"

	LangTc = "tc" // 繁体中文
	LangSc = "sc" // 简体中文
	LangEn = "en" // 英文

	HongKongObservatoryTc = "香港天文台"
	HongKongObservatoryEn = "Hong Kong Observatory"
)

// HKWeather 香港天气预报
// 聚合本港地区天气报告和本港地区天气预报两个接口的数据，返回香港天文台地区天气情况
// 本港地区天气预报：
// https://data.weather.gov.hk/weatherAPI/opendata/weather.php?dataType=flw&lang=tc
// 本港地区天气预报：
// https://data.weather.gov.hk/weatherAPI/opendata/weather.php?dataType=rhrread&lang=tc
func HKWeather(ctx context.Context, lang string) (resp *WeatherResponse, err error) {
	resp = &WeatherResponse{}

	if lang == "" {
		lang = LangTc
	}

	client := guzzle.New(nil, nil)

	//本港地区天气预报
	result, err := client.Get(ctx, HKWeatherApi, map[string]string{"dataType": "flw", "lang": lang}, nil)
	if err != nil {
		return
	}
	var flwResp WeatherFlwResponse
	if err = json.Unmarshal(result.Body, &flwResp); err != nil {
		return
	}

	//本港地区天气报告
	result, err = client.Get(ctx, HKWeatherApi, map[string]string{"dataType": "rhrread", "lang": lang}, nil)
	if err != nil {
		return
	}
	var rhrreadResp WeatherRhrreadResponse
	if err = json.Unmarshal(result.Body, &rhrreadResp); err != nil {
		return
	}

	resp = format(flwResp, rhrreadResp)
	return
}

func format(flwResp WeatherFlwResponse, rhrreadResp WeatherRhrreadResponse) (resp *WeatherResponse) {
	resp = &WeatherResponse{
		Icon:           rhrreadResp.Icon,
		IconLink:       make([]string, 0),
		Temperature:    0,
		Humidity:       0,
		ForecastPeriod: flwResp.ForecastPeriod,
		ForecastDesc:   flwResp.ForecastDesc,
	}

	//图标链接
	for _, iconID := range resp.Icon {
		resp.IconLink = append(resp.IconLink, fmt.Sprintf(HKWeatherIconLink, iconID))
	}

	//温度（取香港天文台）
	for _, item := range rhrreadResp.Temperature.Data {
		resp.Temperature = item.Value
		if item.Place == HongKongObservatoryTc || item.Place == HongKongObservatoryEn {
			break
		}
	}

	//湿度（取香港天文台）
	for _, item := range rhrreadResp.Humidity.Data {
		resp.Humidity = item.Value
		if item.Place == HongKongObservatoryTc || item.Place == HongKongObservatoryEn {
			break
		}
	}

	return
}
