package database

type User struct {
	UUID                         string `json:"uuid" bson:"uuid"`
	Email                        string `json:"email" bson:"email"`
	Password                     string `json:"password" bson:"password"`
	TokenExpirationTimeInSeconds int    `json:"token_expiration_time_in_seconds" bson:"token_expiration_time_in_seconds"`
	RegistrationDateTime         string `json:"registration_date_time" bson:"registration_date_time"`
	UpdateDateTime               string `json:"update_date_time" bson:"update_date_time"`
}

type OrderCommonInfo struct {
	Fee                float64             `json:"fee" bson:"fee"`
	Pairs              []string            `json:"pairs" bson:"pairs"`
	Chains             []string            `json:"chains" bson:"chains"`
	XelisWalletAddress string              `json:"xelis_wallet_address" bson:"xelis_wallet_address"`
	UsdtWalletAddress  string              `json:"usdt_wallet_address" bson:"usdt_wallet_address"`
	UsdcWalletAddress  string              `json:"usdc_wallet_address" bson:"usdc_wallet_address"`
	DepositTimeout     int                 `json:"deposit_timeout" bson:"deposit_timeout"`
	FeePayerTypes      map[string]struct{} `json:"fee_payer_types"`
	Types              map[string]struct{} `json:"types"`
	Networks           map[string]struct{} `json:"networks"`
	Visibility         map[string]struct{} `json:"visibilities"`
	Statuses           map[string]struct{} `json:"statuses"`
}

type OrderData struct {
	ID string `json:"id,omitempty" bson:"id"`
	*Order
	UserUUID         string `json:"user_uuid" bson:"user_uuid"`
	CreationDateTime string `json:"create_date_time" bson:"creation_date_time"`
	UpdateDateTime   string `json:"update_date_time" bson:"update_date_time"`
}

type Order struct {
	Type         string  `json:"type" bson:"type"`
	Pair         string  `json:"pair" bson:"pair"`
	Amount       float64 `json:"amount" bson:"amount"`
	Price        float64 `json:"price" bson:"price"`
	FeePayerType string  `json:"fee_payer_type" bson:"fee_payer_type"`
	Chain        string  `json:"chain" bson:"chain"`
	Network      string  `json:"network" bson:"network"`
	Visibility   string  `json:"visibility" bson:"visibility"`
	Referral     string  `json:"referral" bson:"referral"`
	Status       string  `json:"status" bson:"status"`
}

type OrdererParticipantWallet struct {
	OrderID                         string `json:"order_id" bson:"order_id"`
	OrdererParticipantWalletAddress string `json:"order_participant_wallet_address" bson:"order_participant_wallet_address"`
}
