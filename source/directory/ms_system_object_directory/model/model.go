package model

type Object struct {
	Id                int64   `gorm:"column:id;primaryKey" json:"id"`
	Name              string  `gorm:"column:name" json:"name"`
	Description       string  `gorm:"column:description" json:"description"`
	Status            int64   `gorm:"column:status" json:"status"`
	Prefix            int16   `gorm:"column:prefix" json:"prefix"`
	PrefixNumber      string  `gorm:"column:prefix_number" json:"prefix_number"`
	Level             int16   `gorm:"column:level" json:"level"`
	IsGroup           bool    `gorm:"column:is_group" json:"is_group"`
	IdGroup           *int64  `gorm:"column:id_group" json:"id_group"`
	Ref               *string `gorm:"column:ref" json:"ref"`
	OutputOrder       int16   `gorm:"column:output_order" json:"output_order"`
	Icon              *string `gorm:"column:icon" json:"icon"`
	ObjectTypeName    *string `gorm:"column:object_type_name" json:"object_type_name"`
	ObjectStatusName  *string `gorm:"column:object_status_name" json:"object_status_name"`
	TableColumns      string  `gorm:"column:table_columns" json:"table_columns"`
	TableTitle        string  `gorm:"column:table_title" json:"table_title"`
	ItemTitle         string  `gorm:"column:item_title" json:"item_title"`
	ItemFields        string  `gorm:"column:item_fields" json:"item_fields"`
	OnSaveNew         *string `gorm:"column:on_save_new" json:"on_save_new"`
	OnSaveEdit        *string `gorm:"column:on_save_edit" json:"on_save_edit"`
	OnAcceptance      *string `gorm:"column:on_acceptance" json:"on_acceptance"`
	OnDelete          *string `gorm:"column:on_delete" json:"on_delete"`
	Privileges        string  `gorm:"column:privileges" json:"privileges"`
	RelatedDocuments  *string `gorm:"column:related_documents" json:"related_documents"`
	BusinessProcesses string  `gorm:"column:business_processes" json:"business_processes"`
	PrintingTemplate  string  `gorm:"column:printing_template" json:"printing_template"`
	IdTransaction     *string `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
