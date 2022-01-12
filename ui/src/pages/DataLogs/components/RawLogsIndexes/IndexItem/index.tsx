import indexItemStyles from "@/pages/DataLogs/components/RawLogsIndexes/IndexItem/index.less";
import classNames from "classnames";
import { useEffect, useState } from "react";
import { useModel } from "@@/plugin-model/useModel";
import { Progress, Spin, Tooltip } from "antd";
import useRequest from "@/hooks/useRequest/useRequest";
import api, { IndexDetail } from "@/services/dataLogs";
type IndexItemProps = {
  index: string;
  isActive: boolean;
};
const IndexItem = (props: IndexItemProps) => {
  const { index, isActive } = props;
  const { currentDatabase, currentLogLibrary, startDateTime, endDateTime } =
    useModel("dataLogs");
  const getIndexDetails = useRequest(api.getIndexDetail, {
    loadingText: false,
  });
  const [details, setDetails] = useState<IndexDetail[]>([]);

  useEffect(() => {
    if (
      !isActive ||
      !currentDatabase ||
      !currentLogLibrary ||
      !startDateTime ||
      !endDateTime
    )
      return;
    const params = {
      dt: currentDatabase.datasourceType,
      in: currentDatabase.instanceName,
      db: currentDatabase.databaseName,
      table: currentLogLibrary,
      field: index,
      st: startDateTime,
      et: endDateTime,
    };
    getIndexDetails.run(params).then((res) => {
      if (res?.code === 0) {
        setDetails(res.data);
      }
    });
  }, [index, isActive]);
  return (
    <div className={classNames(indexItemStyles.indexItemMain)}>
      <Spin spinning={getIndexDetails.loading} tip={"加载中..."}>
        <div className={indexItemStyles.detailContextMain}>
          {details.length > 0 ? (
            <>
              {details.map((detail) => (
                <div className={indexItemStyles.context}>
                  <div className={indexItemStyles.title}>
                    <span className={indexItemStyles.name}>
                      <Tooltip title={detail.indexName} placement={"left"}>
                        {detail.indexName}
                      </Tooltip>
                    </span>
                  </div>
                  <div>
                    <Tooltip title={detail.count} placement={"right"}>
                      <Progress
                        className={indexItemStyles.progress}
                        percent={detail.percent}
                        trailColor={"hsla(210, 14%, 83%, 0.4)"}
                        size="small"
                      />
                    </Tooltip>
                  </div>
                </div>
              ))}
            </>
          ) : (
            <span>暂无数据</span>
          )}
        </div>
      </Spin>
    </div>
  );
};
export default IndexItem;
