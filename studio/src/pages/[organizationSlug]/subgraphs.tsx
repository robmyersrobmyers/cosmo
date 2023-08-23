import { EmptyState } from "@/components/empty-state";
import { getDashboardLayout } from "@/components/layout/dashboard-layout";
import { SubgraphsTable } from "@/components/subgraphs-table";
import { Button } from "@/components/ui/button";
import { Loader } from "@/components/ui/loader";
import { NextPageWithLayout } from "@/lib/page";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import { useQuery } from "@tanstack/react-query";
import { getSubgraphs } from "@wundergraph/cosmo-connect/dist/platform/v1/platform-PlatformService_connectquery";
import { EnumStatusCode } from "@wundergraph/cosmo-connect/dist/common_pb";

const SubgraphsDashboardPage: NextPageWithLayout = () => {
  const { data, isLoading, error, refetch } = useQuery(getSubgraphs.useQuery());

  if (isLoading) return <Loader fullscreen />;

  if (error || data.response?.code !== EnumStatusCode.OK)
    return (
      <EmptyState
        icon={<ExclamationTriangleIcon />}
        title="Could not retrieve subgraphs"
        description={
          data?.response?.details || error?.message || "Please try again"
        }
        actions={<Button onClick={() => refetch()}>Retry</Button>}
      />
    );

  if (!data?.graphs) return null;

  return <SubgraphsTable subgraphs={data.graphs} />;
};

SubgraphsDashboardPage.getLayout = (page) => {
  return getDashboardLayout(page, "Subgraphs", "View all your subgraphs");
};

export default SubgraphsDashboardPage;
