import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AppsTable } from "@/components/apps-table";

describe("AppsTable", () => {
  it("renders app rows", () => {
    render(
      <AppsTable
        apps={[
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
