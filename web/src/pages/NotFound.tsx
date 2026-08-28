import { Link } from "react-router-dom";
import { Button, EmptyState } from "@/components/ui";

export function NotFound() {
  return (
    <div className="pt-16">
      <EmptyState
        title="Off the Kessel Run"
        hint="That page doesn't exist."
        action={
          <Link to="/">
            <Button>Back to dashboard</Button>
          </Link>
        }
      />
    </div>
  );
}
