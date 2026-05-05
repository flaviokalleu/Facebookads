export interface Imovel {
  id: string
  user_id: string
  nome: string
  segmento: 'mcmv' | 'medio' | 'alto' | 'comercial' | 'terreno' | 'lancamento'
  cidade: string
  bairro: string
  preco_min?: number
  preco_max?: number
  quartos?: number
  area_m2?: number
  tipologia?: 'apartamento' | 'casa' | 'terreno' | 'sala' | 'galpao' | ''
  diferenciais: string[]
  fotos: string[]
  whatsapp_destino: string
  link_landing: string
  status: 'rascunho' | 'ativo' | 'pausado' | 'vendido'
  created_at: string
  updated_at: string
}

export type ImovelInput = Omit<Imovel, 'id' | 'user_id' | 'created_at' | 'updated_at'>

export const SEGMENTO_LABELS: Record<Imovel['segmento'], string> = {
  mcmv: 'MCMV',
  medio: 'Médio padrão',
  alto: 'Alto padrão',
  comercial: 'Comercial',
  terreno: 'Terreno',
  lancamento: 'Lançamento',
}
export const TIPOLOGIA_LABELS: Record<NonNullable<Imovel['tipologia']>, string> = {
  apartamento: 'Apartamento',
  casa: 'Casa',
  terreno: 'Terreno',
  sala: 'Sala',
  galpao: 'Galpão',
  '': '—',
}
export const STATUS_LABELS: Record<Imovel['status'], string> = {
  rascunho: 'Rascunho',
  ativo: 'Ativo',
  pausado: 'Pausado',
  vendido: 'Vendido',
}

export function useImoveis() {
  const api = useApi()

  async function list(): Promise<Imovel[]> {
    try {
      const res = await api.get<{ data: Imovel[] }>('/imoveis')
      return res?.data || []
    } catch {
      return []
    }
  }

  async function get(id: string): Promise<Imovel | null> {
    try {
      const res = await api.get<{ data: Imovel }>(`/imoveis/${id}`)
      return res?.data || null
    } catch {
      return null
    }
  }

  async function create(input: Partial<ImovelInput>): Promise<Imovel> {
    const res = await api.post<{ data: Imovel }>('/imoveis', input)
    return res.data
  }

  async function update(id: string, input: Partial<ImovelInput>): Promise<Imovel> {
    const res = await api.patch<{ data: Imovel }>(`/imoveis/${id}`, input)
    return res.data
  }

  async function remove(id: string): Promise<void> {
    await api.del(`/imoveis/${id}`)
  }

  return { list, get, create, update, remove }
}
