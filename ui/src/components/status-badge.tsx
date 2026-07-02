import { Badge } from './ui/badge';

export function StatusBadge({ status }: { status?: string }) {
  const value = status || 'completed';
  const variant = value === 'failed' ? 'destructive' : value === 'pending' ? 'warning' : 'success';
  return (
    <Badge variant={variant} className="capitalize">
      {value}
    </Badge>
  );
}
