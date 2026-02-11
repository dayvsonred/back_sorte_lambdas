from pathlib import Path

import duckdb
import pandas as pd
import streamlit as st

BASE_DIR = Path(__file__).resolve().parent

st.set_page_config(page_title="Painel Core", layout="wide")
st.title("Painel de Dados - DynamoDB Export")

default_db = BASE_DIR / "data" / "dashboard.duckdb"
db_path = st.sidebar.text_input("DuckDB path", str(default_db))

if not Path(db_path).exists():
    st.error(f"Arquivo nao encontrado: {db_path}")
    st.stop()


def q(sql: str) -> pd.DataFrame:
    with duckdb.connect(db_path, read_only=True) as con:
        return con.execute(sql).df()


users_total = q("SELECT COUNT(*) AS n FROM users").iloc[0]["n"]
donations_total = q("SELECT COUNT(*) AS n FROM donations").iloc[0]["n"]
payments_total = q(
    """
    SELECT COALESCE(SUM(finalized_amount), 0) AS total
    FROM daily_payments
    """
).iloc[0]["total"]
access_total = q("SELECT COALESCE(SUM(access_events), 0) AS total FROM daily_accesses").iloc[
    0
]["total"]

c1, c2, c3, c4 = st.columns(4)
c1.metric("Usuarios criados", int(users_total))
c2.metric("Doacoes criadas", int(donations_total))
c3.metric("Valor pago finalizado", f"R$ {float(payments_total):,.2f}")
c4.metric("Eventos de acesso", int(access_total))

st.subheader("Usuarios criados por dia")
daily_users = q("SELECT * FROM daily_users")
if daily_users.empty:
    st.info("Sem dados de usuarios.")
else:
    st.line_chart(daily_users.set_index("day")["users_created"])
    st.caption("Detalhamento com e-mails dos usuarios criados")
    users_by_day = q(
        """
        SELECT
          CAST(created_at AS DATE) AS day,
          user_id,
          name,
          email,
          created_at
        FROM users
        WHERE created_at IS NOT NULL
        ORDER BY day DESC, created_at DESC
        """
    )
    selected_day = st.selectbox(
        "Filtrar usuarios por dia",
        options=["Todos"] + [str(d) for d in daily_users["day"].sort_values(ascending=False)],
        index=0,
    )
    if selected_day != "Todos":
        users_by_day = users_by_day[users_by_day["day"].astype(str) == selected_day]
    st.dataframe(users_by_day, use_container_width=True)

st.subheader("Doacoes criadas por dia")
daily_don = q("SELECT * FROM daily_donations")
if daily_don.empty:
    st.info("Sem dados de doacoes.")
else:
    st.line_chart(daily_don.set_index("day")["donations_created"])
    st.caption("Detalhamento das doacoes (nome, link e metadados)")
    donation_details = q(
        """
        SELECT
          day,
          donation_id,
          donation_name,
          user_id,
          amount_declared,
          active,
          closed,
          link_name,
          link_url,
          created_at
        FROM donation_details
        """
    )
    selected_donation_day = st.selectbox(
        "Filtrar doacoes por dia",
        options=["Todos"] + [str(d) for d in daily_don["day"].sort_values(ascending=False)],
        index=0,
    )
    if selected_donation_day != "Todos":
        donation_details = donation_details[
            donation_details["day"].astype(str) == selected_donation_day
        ]
    st.dataframe(donation_details, use_container_width=True)

st.subheader("Acessos diarios")
daily_access = q("SELECT * FROM daily_accesses")
if daily_access.empty:
    st.info("Sem dados de acessos.")
else:
    st.dataframe(daily_access, use_container_width=True)
    st.bar_chart(
        daily_access.set_index("day")[
            ["access_events", "donation_page_accesses", "page_click_events"]
        ]
    )

st.subheader("Explosao de telas acessadas")
daily_break = q("SELECT * FROM daily_access_breakdown")
if daily_break.empty:
    st.info("Sem dados detalhados de acessos.")
else:
    st.dataframe(daily_break, use_container_width=True)
    st.bar_chart(
        daily_break.set_index("day")[
            [
                "acesse_donation",
                "create_pag1",
                "create_pag2",
                "create_pag3",
                "create_pix",
                "create_cartao",
                "create_paypal",
                "create_google",
            ]
        ]
    )

    st.caption("Top campanhas por volume de eventos de acesso")
    by_donation = q("SELECT * FROM access_by_donation LIMIT 20")
    st.dataframe(by_donation, use_container_width=True)

st.subheader("Doacoes pagas por dia")
daily_pay = q("SELECT * FROM daily_payments")
if daily_pay.empty:
    st.info("Sem dados de pagamentos.")
else:
    st.dataframe(daily_pay, use_container_width=True)
    st.line_chart(daily_pay.set_index("day")["finalized_amount"])

st.subheader("Ultimos pagamentos")
latest_pay = q(
    """
    SELECT donation_id, txid, status, finalizado, amount, paid_at, created_at
    FROM payments
    ORDER BY COALESCE(paid_at, created_at) DESC NULLS LAST
    LIMIT 50
    """
)
st.dataframe(latest_pay, use_container_width=True)
