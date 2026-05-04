package domain

import "time"

const (
	ImovelSegmentoMCMV       = "mcmv"
	ImovelSegmentoMedio      = "medio"
	ImovelSegmentoAlto       = "alto"
	ImovelSegmentoComercial  = "comercial"
	ImovelSegmentoTerreno    = "terreno"
	ImovelSegmentoLancamento = "lancamento"

	ImovelStatusRascunho = "rascunho"
	ImovelStatusAtivo    = "ativo"
	ImovelStatusPausado  = "pausado"
	ImovelStatusVendido  = "vendido"
)

type Imovel struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	Nome            string     `json:"nome"`
	Segmento        string     `json:"segmento"`
	Cidade          string     `json:"cidade"`
	Bairro          string     `json:"bairro"`
	PrecoMin        *float64   `json:"preco_min,omitempty"`
	PrecoMax        *float64   `json:"preco_max,omitempty"`
	Quartos         *int       `json:"quartos,omitempty"`
	AreaM2          *float64   `json:"area_m2,omitempty"`
	Tipologia       string     `json:"tipologia"`
	Diferenciais    []string   `json:"diferenciais"`
	Fotos           []string   `json:"fotos"`
	WhatsAppDestino string     `json:"whatsapp_destino"`
	LinkLanding     string     `json:"link_landing"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}
