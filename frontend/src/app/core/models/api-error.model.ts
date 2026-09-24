/**
 * Detalle de un error de validación de un campo específico.
 * Corresponde a domain.ValidationError del backend.
 */
export interface ValidationErrorDetail {
  readonly field: string;
  readonly message: string;
}

/**
 * Formato uniforme de error que devuelve la API.
 *
 * Todos los errores del backend siguen este contrato:
 *   { "error": "...", "details": [...] }
 *
 * Nota sobre `details`: solo aparece en errores de validación (400).
 * En 404, 409 y 500 viene ausente.
 */
export interface ApiError {
  readonly error: string;
  readonly details?: readonly ValidationErrorDetail[];
}

/**
 * Type guard: comprueba si un valor tiene la forma de un ApiError.
 * Útil para estrechar el tipo `unknown` de las respuestas HTTP.
 *
 * En Angular 21, HttpErrorResponse.error es de tipo `any` o `unknown`
 * según la configuración. Este guard nos ayuda a tratarlo con seguridad.
 */
export function isApiError(value: unknown): value is ApiError {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const obj = value as Record<string, unknown>;
  return typeof obj['error'] === 'string';
}