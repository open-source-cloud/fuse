-- Phase 2.5: LISTEN/NOTIFY triggers for real-time state synchronization.
-- Nodes subscribe to the 'workflow_state_change' channel for fast workflow pickup.

CREATE OR REPLACE FUNCTION notify_workflow_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('workflow_state_change',
        NEW.workflow_id || ':' || NEW.state::TEXT || ':' || COALESCE(NEW.claimed_by, '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_workflow_state
    AFTER INSERT OR UPDATE OF state, claimed_by ON workflows
    FOR EACH ROW EXECUTE FUNCTION notify_workflow_change();
