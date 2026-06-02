import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PackagesTable } from "@/components/packages-table";

describe("PackagesTable", () => {
  it("renders package rows", () => {
    render(
      <PackagesTable
        packages={[
          {
            id: "nginx",
            name: "nginx",
            description: "Web server",
            category: "web",
            installed: true,
            dependencies_satisfied: true,
          },
        ]}
      />,
    );

    expect(screen.getByText("nginx")).toBeInTheDocument();
    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByText("Web server")).toBeInTheDocument();
  });
});
