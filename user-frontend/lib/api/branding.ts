export interface BrandingSettings {
  site_name: string;
  logo_url: string;
  primary_color: string;
  footer_text: string;
  announcement: string;
  theme: string;
}

export async function getBrandingSettings(): Promise<BrandingSettings> {
  const response = await fetch('/api/v1/branding');
  if (!response.ok) {
    throw new Error('Failed to fetch branding settings');
  }
  return response.json();
}
