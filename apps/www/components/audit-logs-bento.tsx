import { AuditLogs } from "@/components/audit/audit-logs";

export function AuditLogsBento() {
  return (
    <div className="relative group no-scrollbar overflow-hidden w-full xl:mt-10  border-[.75px] h-[520px] rounded-[32px] border-[#ffffff]/10 flex overflow-x-hidden bg-gradient-to-br from-white/10 to-black">
      <AuditLogs className=" sm:h-[400px] w-full sm:ml-[40px]" />
      <div className="absolute inset-0 w-full h-full duration-500 pointer-events-none bg-gradient-to-tr from-black via-black/40 to-black/0 group-hover:opacity-0 group-hover:backdrop-blur-0" />
      <div className="duration-500 group-hover:opacity-0 group-hover:pointer-events-none">
        <AuditLogsText />
      </div>
    </div>
  );
}

export function AuditLogsText() {
  return (
    <div className="flex flex-col text-white absolute left-[20px] sm:left-[40px] xl:left-[40px] bottom-[40px] max-w-[350px]">
      <div className="flex items-center w-full">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
          fill="none"
        >
          <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M6.5 3H6.47812C5.7966 3 5.25458 2.99999 4.81729 3.03572C4.36949 3.07231 3.98765 3.14884 3.63803 3.32698C3.07354 3.6146 2.6146 4.07354 2.32698 4.63803C2.14884 4.98765 2.07231 5.36949 2.03572 5.81729C1.99999 6.25458 2 6.7966 2 7.47812V7.5V15.5V15.5219C2 16.2034 1.99999 16.7454 2.03572 17.1827C2.07231 17.6305 2.14884 18.0123 2.32698 18.362C2.6146 18.9265 3.07354 19.3854 3.63803 19.673C3.98765 19.8512 4.36949 19.9277 4.81729 19.9643C5.25457 20 5.79657 20 6.47806 20H6.47811H6.5H17.5H17.5219H17.5219C18.2034 20 18.7454 20 19.1827 19.9643C19.6305 19.9277 20.0123 19.8512 20.362 19.673C20.9265 19.3854 21.3854 18.9265 21.673 18.362C21.8512 18.0123 21.9277 17.6305 21.9643 17.1827C22 16.7454 22 16.2034 22 15.5219V15.5219V15.5V7.5V7.47811V7.47806C22 6.79657 22 6.25457 21.9643 5.81729C21.9277 5.36949 21.8512 4.98765 21.673 4.63803C21.3854 4.07354 20.9265 3.6146 20.362 3.32698C20.0123 3.14884 19.6305 3.07231 19.1827 3.03572C18.7454 2.99999 18.2034 3 17.5219 3H17.5H6.5ZM4.09202 4.21799C4.27717 4.12365 4.51276 4.06393 4.89872 4.0324C5.29052 4.00039 5.79168 4 6.5 4H17.5C18.2083 4 18.7095 4.00039 19.1013 4.0324C19.4872 4.06393 19.7228 4.12365 19.908 4.21799C20.2843 4.40973 20.5903 4.7157 20.782 5.09202C20.8764 5.27717 20.9361 5.51276 20.9676 5.89872C20.9996 6.29052 21 6.79168 21 7.5V15.5C21 16.2083 20.9996 16.7095 20.9676 17.1013C20.9361 17.4872 20.8764 17.7228 20.782 17.908C20.5903 18.2843 20.2843 18.5903 19.908 18.782C19.7228 18.8764 19.4872 18.9361 19.1013 18.9676C18.7095 18.9996 18.2083 19 17.5 19H6.5C5.79168 19 5.29052 18.9996 4.89872 18.9676C4.51276 18.9361 4.27717 18.8764 4.09202 18.782C3.7157 18.5903 3.40973 18.2843 3.21799 17.908C3.12365 17.7228 3.06393 17.4872 3.0324 17.1013C3.00039 16.7095 3 16.2083 3 15.5V7.5C3 6.79168 3.00039 6.29052 3.0324 5.89872C3.06393 5.51276 3.12365 5.27717 3.21799 5.09202C3.40973 4.7157 3.7157 4.40973 4.09202 4.21799ZM10.8841 6.82058L8.38411 9.82058L8.0336 10.2412L7.64645 9.85404L6.14645 8.35404L6.85355 7.64693L7.9664 8.75978L10.1159 6.1804L10.8841 6.82058ZM10.8841 12.8206L8.38411 15.8206L8.0336 16.2412L7.64645 15.854L6.14645 14.354L6.85355 13.6469L7.9664 14.7598L10.1159 12.1804L10.8841 12.8206ZM12.9995 9.00049H17.9995V8.00049H12.9995V9.00049ZM17.9995 15.0005H12.9995V14.0005H17.9995V15.0005Z"
            fill="white"
            fillOpacity="0.4"
          />
        </svg>
        <h3 className="ml-4 text-lg font-medium text-white">Audit Logs</h3>
      </div>
      <p className="mt-4 leading-6 text-white/60">
        Audit logs out of the box. Focus on building your product and let us handle security and
        compliance.
      </p>
    </div>
  );
}
