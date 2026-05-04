export interface AdAccount {
  id: string | number
  meta_id?: string
  name: string
  currency?: string
  account_status?: number
}

export interface BusinessManager {
  id: string | number
  meta_id?: string
  name: string
  verification_status?: string
  ad_accounts?: AdAccount[]
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
