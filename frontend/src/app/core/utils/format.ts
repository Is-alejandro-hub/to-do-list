import type { Priority } from '@core/models';

/**
 * Etiqueta en español para cada prioridad.
 * Es una decisión de UI: el backend usa códigos, el usuario ve palabras.
 */
export function priorityLabel(p: Priority): string {
  switch (p) {
    case 'HIGH':
      return 'Alta';
    case 'MEDIUM':
      return 'Media';
    case 'LOW':
      return 'Baja';
  }
}

/**
 * Clase CSS para el badge según la prioridad.
 * Devuelve el nombre de clase que aplicaremos al elemento.
 */
export function priorityClass(p: Priority): string {
  return `badge badge--${p.toLowerCase()}`;
}

/**
 * Formatea una fecha ISO 8601 al formato "dd MMM yyyy, HH:mm".
 * Usa la API Intl nativa del navegador; no depende de librerías externas.
 */
export function formatDateTime(iso: string): string {
  const date = new Date(iso);
  return new Intl.DateTimeFormat('es-MX', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

/**
 * Formatea solo la fecha (sin hora).
 */
export function formatDate(iso: string): string {
  const date = new Date(iso);
  return new Intl.DateTimeFormat('es-MX', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(date);
}

/**
 * ¿La fecha dada está en el pasado?
 */
export function isPast(iso: string | null): boolean {
  if (!iso) return false;
  return new Date(iso).getTime() < Date.now();
}
import type { AuditAction } from '@core/models';

/**
 * Etiqueta en español para cada acción de auditoría.
 */
export function auditActionLabel(action: AuditAction): string {
  switch (action) {
    case 'INSERT':      return 'Creada';
    case 'UPDATE':      return 'Actualizada';
    case 'SOFT_DELETE': return 'Eliminada';
    case 'RESTORE':     return 'Restaurada';
    case 'DELETE':      return 'Eliminada permanentemente';
  }
}

/**
 * Clase CSS para el badge según la acción.
 */
export function auditActionClass(action: AuditAction): string {
  return `audit-badge audit-badge--${action.toLowerCase().replace('_', '-')}`;
}