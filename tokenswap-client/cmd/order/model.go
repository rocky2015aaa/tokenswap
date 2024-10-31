package order

type OrderRequest struct {
	*Order
	OrdererWalletAddress string `json:"orderer_wallet_address"`
	Password             string `json:"password"`
}

type Order struct {
	Type         string  `json:"type"`
	Pair         string  `json:"pair"`
	Amount       float64 `json:"amount"`
	Price        float64 `json:"price"`
	FeePayerType string  `json:"fee_payer_type"`
	Chain        string  `json:"chain"`
	Network      string  `json:"network"`
	Visibility   string  `json:"visibility"`
	Referral     string  `json:"referral"`
	Status       string  `json:"status"`
}

type OrderDetail struct {
	ID string
	*Order
	CreateDateTime string
}

type OrderCommonInfo struct {
	Fee             float64
	Pairs           []string
	Chains          []string
	WalletAddresses map[string]string
	DepositTimeout  int
	FeePayerTypes   map[string]interface{}
	Types           map[string]interface{}
	Networks        map[string]interface{}
	Visibilities    map[string]interface{}
	Statuses        map[string]interface{}
}

type OrderTakeRequest struct {
	Password string `json:"password"`
}

type Config struct {
	DepositTimeout     int    `json:"deposit_timeout"`
	XelisWalletAddress string `json:"xelis_wallet_address"`
}
