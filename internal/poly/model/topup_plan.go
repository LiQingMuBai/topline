package model

type CarrierTopupPlan struct {
	BaseModel
	NameEn    string `json:"name_en" gorm:"column:name_en;size:255" form:"name_en"`
	NameCn    string `json:"name_cn" gorm:"column:name_cn;size:255" form:"name_cn"`
	EventName string `json:"event_name" gorm:"column:event_name;size:255" form:"event_name"`
	Price     string `json:"price" gorm:"column:price;size:255" form:"price"`
	Status    int    `json:"status" gorm:"column:status" form:"status"`
	CountryID int    `json:"country_id" gorm:"column:country_id" form:"country_id"`
	CarrierID int    `json:"carrier_id" gorm:"column:carrier_id" form:"carrier_id"`
}

func (CarrierTopupPlan) TableName() string {
	return "polytopup_carrier_topup_plan"
}

type ExpensesTopupPlan struct {
	BaseModel
	NameEn    string `json:"name_en" gorm:"column:name_en;size:255" form:"name_en"`
	NameCn    string `json:"name_cn" gorm:"column:name_cn;size:255" form:"name_cn"`
	EventName string `json:"event_name" gorm:"column:event_name;size:255" form:"event_name"`
	Price     string `json:"price" gorm:"column:price;size:255" form:"price"`
	Status    int    `json:"status" gorm:"column:status" form:"status"`
	CountryID int    `json:"country_id" gorm:"column:country_id" form:"country_id"`
	CarrierID int    `json:"carrier_id" gorm:"column:carrier_id" form:"carrier_id"`
}

func (ExpensesTopupPlan) TableName() string {
	return "polytopup_topup_plan"
}

type DataTopupPlan struct {
	BaseModel
	NameEn    string `json:"name_en" gorm:"column:name_en;size:255" form:"name_en"`
	NameCn    string `json:"name_cn" gorm:"column:name_cn;size:255" form:"name_cn"`
	EventName string `json:"event_name" gorm:"column:event_name;size:255" form:"event_name"`
	Price     string `json:"price" gorm:"column:price;size:255" form:"price"`
	Status    int    `json:"status" gorm:"column:status" form:"status"`
	CountryID int    `json:"country_id" gorm:"column:country_id" form:"country_id"`
	CarrierID int    `json:"carrier_id" gorm:"column:carrier_id" form:"carrier_id"`
}

func (DataTopupPlan) TableName() string {
	return "polytopup_data_plan"
}

type UserMobile struct {
	BaseModel
	Mobile      string `json:"mobile" gorm:"column:mobile;size:255" form:"mobile"`
	ReminderDay string `json:"reminder_day" gorm:"column:reminder_day;size:255" form:"reminder_day"`
	Status      int    `json:"status" gorm:"column:status" form:"status"`
	CountryID   int    `json:"country_id" gorm:"column:country_id" form:"country_id"`
	ChatID      int64  `json:"chat_id" gorm:"column:chat_id" form:"chat_id"`
}

func (UserMobile) TableName() string {
	return "polytopup_user_mobile"
}
