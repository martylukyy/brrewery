import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PackagesTable } from "@/components/packages-table";

describe("PackagesTable", () => {
  it("renders package rows", () => {
    render(
      <PackagesTable
        packages={[
          {
            id: "sonarr",
            name: "Sonarr",
            description: "TV series management",
            category: "arr",
            installed: true,
            dependencies_satisfied: true,
          },
        ]}
      />,
    );

    expect(screen.getByText("Sonarr")).toBeInTheDocument();
    expect(screen.getByText("Installed")).toBeInTheDocument();
    expect(screen.getByText("TV series management")).toBeInTheDocument();
  });
});
