import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { AppStatus } from "@/lib/api";

type Props = {
  apps: AppStatus[];
};

export function AppsTable({ apps }: Props) {
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <Table className="text-sm">
        <TableHeader>
          <TableRow>
            <TableHead className="px-4 py-3">Name</TableHead>
            <TableHead className="px-4 py-3">Category</TableHead>
            <TableHead className="px-4 py-3">Status</TableHead>
            <TableHead className="px-4 py-3">Description</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {apps.map((app) => (
            <TableRow key={app.id}>
              <TableCell className="px-4 py-3 font-medium text-foreground">{app.name}</TableCell>
              <TableCell className="px-4 py-3 text-muted-foreground">{app.category}</TableCell>
              <TableCell className="px-4 py-3">
                <Badge variant={app.installed ? "secondary" : "outline"}>
                  {app.installed ? "Installed" : "Not installed"}
                </Badge>
              </TableCell>
              <TableCell className="px-4 py-3 text-muted-foreground">{app.description}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
