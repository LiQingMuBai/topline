package model

// Country表 结构体  Country
type Country struct {
	ALI_MODEL
	NameEn    string `json:"name_en" gorm:"column:name_en;size:255" form:"name_en"`
	NameCn    string `json:"name_cn" gorm:"column:name_cn;size:255" form:"name_cn"`
	EventName string `json:"event_name" gorm:"column:event_name;size:255" form:"event_name"`
	Status    int    `json:"status" gorm:"column:status" form:"status"`
}

func (Country) TableName() string {
	return "polytopup_country"
}
