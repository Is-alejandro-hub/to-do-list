/**
 * Genera un UUID v4 usando la API crypto del navegador.
 *
 * `crypto.randomUUID()` está disponible en:
 *   - Chrome 92+
 *   - Firefox 95+
 *   - Safari 15.4+
 *   - Edge 92+
 *
 * Es criptográficamente seguro y libre de colisiones en la práctica.
 * Se usa para la Idempotency-Key del POST /tasks.
 */
export function generateUUID(): string {
  return crypto.randomUUID();
}