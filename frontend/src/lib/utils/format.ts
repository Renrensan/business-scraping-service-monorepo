/**
 * Formats a raw phone string into a direct WhatsApp wa.me link.
 * Handles Indonesian numbers (starting with 0 -> converted to 62),
 * and standard international numbers.
 */
export function getWhatsAppUrl(rawPhone?: string | null): string | null {
  if (!rawPhone) return null;

  // Keep only digits
  let clean = rawPhone.replace(/\D/g, '');

  if (!clean) return null;

  // Handle local Indonesian numbers like 0812... -> 62812...
  if (clean.startsWith('0')) {
    clean = '62' + clean.slice(1);
  }

  // Basic sanity check: valid phone numbers usually have at least 8 digits
  if (clean.length < 8) {
    return null;
  }

  return `https://wa.me/${clean}`;
}
