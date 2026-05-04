export interface AdAccount {
  meta_id: string
  name: string
  currency?: string
  account_status?: number
  balance?: number
  amount_spent?: number
  spend_cap?: number
  access_kind?: string
}

export interface Page {
  meta_id: string
  name: string
  category?: string
  fan_count?: number
}

export interface Pixel {
  meta_id: string
  name: string
  is_active?: boolean
  last_fired?: string
}

export interface BusinessManager {
  meta_id: string
  name: string
  verification_status?: string
  vertical?: string
  accounts?: AdAccount[]
  pages?: Page[]
  pixels?: Pixel[]
}

export interface MetaTree {
  businesses: BusinessManager[]
  personal_accounts: AdAccount[]
}

export function useMetaTree() {
  const api = useApi()

  async function fetchTree(): Promise<MetaTree> {
    try {
      const res = await api.get<{ data: MetaTree }>('/businesses')
      const d = res?.data ?? (res as unknown as MetaTree)
      return {
        businesses: d?.businesses || [],
        personal_accounts: d?.personal_accounts || [],
      }
    } catch {
      return { businesses: [], personal_accounts: [] }
    }
  }

  return { fetchTree }
}
