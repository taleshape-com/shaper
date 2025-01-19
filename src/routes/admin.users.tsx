import { createFileRoute } from "@tanstack/react-router";
import { translate } from "../lib/translate";

export const Route = createFileRoute("/admin/users")({
  component: UsersManagement,
});

function UsersManagement() {
  return (
    <>
      <h2 className="text-xl font-semibold mb-4">
        {translate("User Management")}
      </h2>
      <p className="text-gray-500">
        User management coming soon...
      </p>
    </>
  );
}