-- 添加外键约束（允许已存在）

ALTER TABLE cluster_states
    DROP CONSTRAINT IF EXISTS fk_cluster_states_cluster_id;

ALTER TABLE cluster_states
    ADD CONSTRAINT fk_cluster_states_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE nodes
    DROP CONSTRAINT IF EXISTS fk_nodes_cluster_id;

ALTER TABLE nodes
    ADD CONSTRAINT fk_nodes_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE cluster_resources
    DROP CONSTRAINT IF EXISTS fk_cluster_resources_cluster_id;

ALTER TABLE cluster_resources
    ADD CONSTRAINT fk_cluster_resources_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE events
    DROP CONSTRAINT IF EXISTS fk_events_cluster_id;

ALTER TABLE events
    ADD CONSTRAINT fk_events_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE security_policies
    DROP CONSTRAINT IF EXISTS fk_security_policies_cluster_id;

ALTER TABLE security_policies
    ADD CONSTRAINT fk_security_policies_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;

ALTER TABLE autoscaling_policies
    DROP CONSTRAINT IF EXISTS fk_autoscaling_policies_cluster_id;

ALTER TABLE autoscaling_policies
    ADD CONSTRAINT fk_autoscaling_policies_cluster_id
    FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE;
